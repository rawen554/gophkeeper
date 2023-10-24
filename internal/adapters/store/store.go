package store

import (
	"context"
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
	PutDataRecord(data *models.DataRecord, userID uint64) error
	GetUserRecord(recordID uint64, userID uint64) (*models.DataRecord, error)
	GetUserRecords(userID uint64) ([]models.DataRecord, error)
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

func NewStore(ctx context.Context, dsn string, logLevel string) (Store, error) {
	conn, err := gorm.Open(postgres.New(postgres.Config{
		DSN: dsn,
	}), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("cant connect to db: %w", err)
	}

	if err := prepareConnPool(conn); err != nil {
		return nil, err
	}

	if err := runMigrations(dsn); err != nil {
		return nil, err
	}

	conn.Logger = logger.Default.LogMode(logger.LogLevel(utils.ConvertLogLevelToInt(logLevel)))
	if err := conn.AutoMigrate(&models.User{}, &models.DataRecord{}); err != nil {
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

func (db *DBStore) PutDataRecord(data *models.DataRecord, userID uint64) error {
	result := db.conn.Where("user_id = ?", userID).Save(&data)

	if err := result.Error; err != nil {
		return fmt.Errorf("error saving data: %w", err)
	}

	return nil
}

func (db *DBStore) GetUserRecord(recordID uint64, userID uint64) (*models.DataRecord, error) {
	record := models.DataRecord{}
	result := db.conn.Where(&models.DataRecord{UserID: userID, ID: recordID}).Find(&record)

	if err := result.Error; err != nil {
		return nil, fmt.Errorf("error getting order: %w", err)
	}

	return &record, nil
}

func (db *DBStore) GetUserRecords(userID uint64) ([]models.DataRecord, error) {
	records := make([]models.DataRecord, 0)
	result := db.conn.Where(&models.DataRecord{UserID: userID}).Find(&records)

	if err := result.Error; err != nil {
		return nil, fmt.Errorf("error getting all user orders: %w", err)
	}

	if len(records) == 0 {
		return nil, models.ErrNoData
	}

	return records, nil
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
