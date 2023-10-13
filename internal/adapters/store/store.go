package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rawen554/goph-keeper/internal/models"
	"github.com/rawen554/goph-keeper/internal/utils"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DBStore struct {
	conn *gorm.DB
}

type Store interface {
	CreateUser(user *models.User) (int64, error)
	GetUser(u *models.User) (*models.User, error)
	PutOrder(number string, userID uint64) error
	UpdateOrder(o *models.Order) (int64, error)
	GetUserOrders(userID uint64) ([]models.Order, error)
	GetUnprocessedOrders() ([]models.Order, error)
	GetUserBalance(userID uint64) (*models.UserBalanceShema, error)
	CreateWithdraw(userID uint64, w models.BalanceWithdrawShema) error
	GetWithdrawals(userID uint64) ([]models.Withdraw, error)
	Ping() error
	Close()
}

const (
	MaxIdleConns = 10
	MaxOpenConns = 100
)

var ErrDBInsertConflict = errors.New("conflict insert into table, returned stored value")
var ErrURLDeleted = errors.New("url is deleted")
var ErrLoginNotFound = errors.New("login not found")
var ErrDuplicateLogin = errors.New("login already registered")
var ErrNotEnoughAmount = errors.New("not enough balance")

const connectTick = 5

func NewStore(ctx context.Context, dsn string, logLevel string) (Store, error) {
	conn, err := ConnectLoop(dsn, connectTick*time.Second, time.Minute)
	if err != nil {
		return nil, err
	}

	if err := prepareConnPool(conn); err != nil {
		return nil, err
	}

	if err := runMigrations(dsn); err != nil {
		return nil, err
	}

	conn.Logger = logger.Default.LogMode(logger.LogLevel(utils.ConvertLogLevelToInt(logLevel)))
	if err := conn.AutoMigrate(&models.User{}, &models.Order{}, &models.Withdraw{}); err != nil {
		return nil, fmt.Errorf("error auto migrating models: %w", err)
	}

	log.Println("successfully connected to the database")

	return &DBStore{conn: conn}, nil
}

//go:embed migrations/*.sql
var migrationsDir embed.FS

func runMigrations(dsn string) error {
	d, err := iofs.New(migrationsDir, "migrations")
	if err != nil {
		return fmt.Errorf("failed to return an iofs driver: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", d, dsn)
	if err != nil {
		return fmt.Errorf("failed to get a new migrate instance: %w", err)
	}
	if err := m.Up(); err != nil {
		if !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("failed to apply migrations to the DB: %w", err)
		}
	}
	return nil
}

func prepareConnPool(conn *gorm.DB) error {
	sqlDB, err := conn.DB()
	if err != nil {
		return fmt.Errorf("cannot get interface sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(MaxIdleConns)
	sqlDB.SetMaxOpenConns(MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return nil
}

func ConnectLoop(dsn string, tick time.Duration, timeout time.Duration) (*gorm.DB, error) {
	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	timeoutExceeded := time.After(timeout)

	for {
		select {
		case <-timeoutExceeded:
			return nil, fmt.Errorf("db connection failed after %s timeout", timeout)
		case <-ticker.C:
			conn, err := gorm.Open(postgres.New(postgres.Config{
				DSN: dsn,
			}), &gorm.Config{})
			if err != nil {
				log.Printf("error connecting to db: %v", err)
			} else {
				return conn, nil
			}
		}
	}
}

func (db *DBStore) CreateUser(user *models.User) (int64, error) {
	result := db.conn.Create(user)

	if result.Error != nil {
		var pgErr *pgconn.PgError
		if errors.As(result.Error, &pgErr) {
			if pgErr.Code == pgerrcode.UniqueViolation {
				return 0, ErrDuplicateLogin
			}
		}

		log.Printf("error saving user to db: %v", result.Error)
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func (db *DBStore) GetUser(u *models.User) (*models.User, error) {
	var user models.User
	result := db.conn.Where(u).First(&user)

	if result.RowsAffected == 0 {
		return nil, ErrLoginNotFound
	}

	return &user, result.Error
}

func (db *DBStore) GetUserBalance(userID uint64) (*models.UserBalanceShema, error) {
	var user models.User
	var userBalance models.UserBalanceShema
	result := db.conn.Model(&user).Where(&models.User{ID: userID}).Take(&userBalance)

	return &userBalance, result.Error
}

func (db *DBStore) PutOrder(number string, userID uint64) error {
	var order models.Order
	result := db.conn.
		Where(models.Order{Number: number}).
		Attrs(models.Order{UserID: userID, Status: models.NEW}).
		FirstOrCreate(&order)

	if err := result.Error; err != nil {
		return fmt.Errorf("error saving order: %w", err)
	}

	if order.UserID != userID {
		return models.ErrOrderHasBeenProcessedByAnotherUser
	}

	if order.UserID == userID && order.Number == number && result.RowsAffected == 0 {
		return models.ErrOrderHasBeenProcessedByUser
	}

	return nil
}

func (db *DBStore) UpdateOrder(o *models.Order) (int64, error) {
	result := db.conn.Model(o).Updates(&models.Order{Accrual: o.Accrual, Status: o.Status})
	return result.RowsAffected, result.Error
}

func (db *DBStore) GetUnprocessedOrders() ([]models.Order, error) {
	orders := make([]models.Order, 0)
	result := db.conn.Where(
		"status = @new OR status = @processing",
		sql.Named("new", models.NEW), sql.Named("processing", models.PROCESSING),
	).Find(&orders)

	if err := result.Error; err != nil {
		return nil, fmt.Errorf("error getting all unprocessed orders: %w", err)
	}

	return orders, nil
}

func (db *DBStore) GetUserOrders(userID uint64) ([]models.Order, error) {
	orders := make([]models.Order, 0)
	result := db.conn.Order("uploaded_at asc").Where(&models.Order{UserID: userID}).Find(&orders)

	if err := result.Error; err != nil {
		return nil, fmt.Errorf("error getting all user orders: %w", err)
	}

	if len(orders) == 0 {
		return nil, models.ErrUserHasNoItems
	}

	return orders, nil
}

func (db *DBStore) CreateWithdraw(userID uint64, w models.BalanceWithdrawShema) error {
	u, err := db.GetUser(&models.User{ID: userID})
	if err != nil {
		return fmt.Errorf("cant get user: %w", err)
	}

	if u.Balance < w.Sum {
		return ErrNotEnoughAmount
	}

	u.Balance -= w.Sum
	u.Withdrawn += w.Sum

	err = db.conn.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(u).Error; err != nil {
			return fmt.Errorf("update user balance error: %w", err)
		}

		if err := tx.Create(&models.Withdraw{OrderNum: w.Order, Sum: w.Sum, UserID: userID}).Error; err != nil {
			return fmt.Errorf("create withdraw error: %w", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("withdraw not commited: %w", err)
	}

	return nil
}

func (db *DBStore) GetWithdrawals(userID uint64) ([]models.Withdraw, error) {
	withdrawals := make([]models.Withdraw, 0)
	result := db.conn.Order("processed_at asc").Where(&models.Withdraw{UserID: userID}).Find(&withdrawals)

	if err := result.Error; err != nil {
		return nil, fmt.Errorf("error getting all user withdrawals: %w", err)
	}

	if len(withdrawals) == 0 {
		return nil, models.ErrUserHasNoItems
	}

	return withdrawals, nil
}

func (db *DBStore) Ping() error {
	sqlDB, err := db.conn.DB()
	if err != nil {
		return fmt.Errorf("error getting sql.DB interface: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("lost connection to DB: %w", err)
	}

	return nil
}

func (db *DBStore) Close() {
	sqlDB, err := db.conn.DB()
	if err != nil {
		log.Printf("gorm cant get sql.DB interface: %v", err)
	}

	if err := sqlDB.Close(); err != nil {
		log.Printf("error closing connection to DB: %v", err)
	}
}
