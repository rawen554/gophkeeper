package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/rawen554/goph-keeper/internal/adapters/store"
	"github.com/rawen554/goph-keeper/internal/config"
	"github.com/rawen554/goph-keeper/internal/middleware/auth"
	"github.com/rawen554/goph-keeper/internal/models"
	"github.com/rawen554/goph-keeper/internal/utils"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type App struct {
	config *config.ServerConfig
	store  store.Store
	logger *zap.SugaredLogger
}

const (
	bcryptCost   = 7
	maxCookieAge = 3600 * 24 * 30
)

func NewApp(config *config.ServerConfig, store store.Store, logger *zap.SugaredLogger) *App {
	return &App{
		config: config,
		store:  store,
		logger: logger,
	}
}

func (a *App) NewServer() (*http.Server, error) {
	r, err := a.SetupRouter()
	if err != nil {
		return nil, fmt.Errorf("error init router: %w", err)
	}

	return &http.Server{
		Addr:    a.config.RunAddr,
		Handler: r,
	}, nil
}

func (a *App) Login(c *gin.Context) {
	req := c.Request
	res := c.Writer

	userCreds := models.User{}
	if err := json.NewDecoder(req.Body).Decode(&userCreds); err != nil {
		a.logger.Errorf("user credentials cannot be decoded: %v", err)
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	userReq := models.User{
		Login:    userCreds.Login,
		Password: userCreds.Password,
	}

	u, err := a.store.GetUser(&models.User{Login: userReq.Login})
	if err != nil {
		if errors.Is(err, store.ErrLoginNotFound) {
			a.logger.Errorf("login not found: %v", err)
			res.WriteHeader(http.StatusUnauthorized)
			return
		} else {
			a.logger.Errorf("cannot get user: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(userReq.Password)); err != nil {
		res.WriteHeader(http.StatusUnauthorized)
		return
	}
	userReq.ID = u.ID

	jwt, err := auth.BuildJWTString(userReq.ID, a.config.Key)
	if err != nil {
		a.logger.Errorf("cannot build jwt string for authorized user: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	c.SetCookie(auth.CookieName, jwt, maxCookieAge, "", "", false, true)
	res.WriteHeader(http.StatusOK)
}

func (a *App) Register(c *gin.Context) {
	req := c.Request
	res := c.Writer

	userCreds := models.User{}
	if err := json.NewDecoder(req.Body).Decode(&userCreds); err != nil {
		a.logger.Errorf("body cannot be decoded: %v", err)
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	userReq := models.User{
		Login:    userCreds.Login,
		Password: userCreds.Password,
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(userReq.Password), bcryptCost) //TODO: check sha512, cryptoready
	if err != nil {
		a.logger.Errorf("cannot hash pass: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	userReq.Password = string(hash)

	if _, err = a.store.CreateUser(&userReq); err != nil {
		if errors.Is(err, store.ErrDuplicateLogin) {
			a.logger.Errorf("login already taken: %v", err)
			res.WriteHeader(http.StatusConflict)
			return
		} else {
			a.logger.Errorf("cannot operate user creds: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	jwt, err := auth.BuildJWTString(userReq.ID, a.config.Key)
	if err != nil {
		a.logger.Errorf("cannot build jwt string: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	c.SetCookie(auth.CookieName, jwt, maxCookieAge, "", "", false, true)
	res.WriteHeader(http.StatusOK)
}

func (a *App) PutOrder(c *gin.Context) {
	userID := c.GetUint64(auth.UserIDKey.ToString())
	req := c.Request
	res := c.Writer
	if userID == 0 {
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	number := string(body)

	if isValidLuhn := utils.IsValidLuhn(number); !isValidLuhn {
		res.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	if err := a.store.PutOrder(number, userID); err != nil {
		switch {
		case errors.Is(err, models.ErrOrderHasBeenProcessedByAnotherUser):
			res.WriteHeader(http.StatusConflict)
			return

		case errors.Is(err, models.ErrOrderHasBeenProcessedByUser):
			res.WriteHeader(http.StatusOK)
			return

		default:
			a.logger.Errorf("unhandled error: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	res.WriteHeader(http.StatusAccepted)
}

//nolint:dupl // code deduplication will lead to bad code extending in future
func (a *App) GetOrders(c *gin.Context) {
	userID := c.GetUint64(auth.UserIDKey.ToString())
	res := c.Writer
	if userID == 0 {
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	orders, err := a.store.GetUserOrders(userID)
	if err != nil {
		if errors.Is(err, models.ErrUserHasNoItems) {
			res.WriteHeader(http.StatusNoContent)
			return
		}

		a.logger.Errorf("error getting user orders: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, orders)
}

//nolint:dupl // code deduplication will lead to bad code extending in future
func (a *App) GetWithdrawals(c *gin.Context) {
	userID := c.GetUint64(auth.UserIDKey.ToString())
	res := c.Writer
	if userID == 0 {
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	withdrawals, err := a.store.GetWithdrawals(userID)
	if err != nil {
		if errors.Is(err, models.ErrUserHasNoItems) {
			res.WriteHeader(http.StatusNoContent)
			return
		}

		a.logger.Errorf("unknown error: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, withdrawals)
}

func (a *App) GetBalance(c *gin.Context) {
	userID := c.GetUint64(auth.UserIDKey.ToString())
	res := c.Writer
	if userID == 0 {
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	balance, err := a.store.GetUserBalance(userID)
	if err != nil {
		a.logger.Errorf("Error getting user balance: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK)
	res.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(res).Encode(balance); err != nil {
		a.logger.Errorf("Error writing response in JSON: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (a *App) BalanceWithdraw(c *gin.Context) {
	userID := c.GetUint64(auth.UserIDKey.ToString())
	res := c.Writer
	req := c.Request
	if userID == 0 {
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	var withdrawRequest models.BalanceWithdrawShema
	if err := json.NewDecoder(req.Body).Decode(&withdrawRequest); err != nil {
		a.logger.Errorf("Body cannot be decoded: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	if isValidLuhn := utils.IsValidLuhn(withdrawRequest.Order); !isValidLuhn {
		res.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	if err := a.store.CreateWithdraw(userID, withdrawRequest); err != nil {
		if errors.Is(err, store.ErrNotEnoughAmount) {
			res.WriteHeader(http.StatusPaymentRequired)
			return
		}
		a.logger.Errorf("cant save withdraw: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK)
}

func (a *App) Ping(c *gin.Context) {
	if err := a.store.Ping(); err != nil {
		a.logger.Errorf("Error opening connection to DB: %v", err)
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	c.Writer.WriteHeader(http.StatusOK)
}
