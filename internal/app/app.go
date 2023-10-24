package app

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/rawen554/goph-keeper/internal/adapters/store"
	"github.com/rawen554/goph-keeper/internal/config"
	"github.com/rawen554/goph-keeper/internal/middleware/auth"
	"github.com/rawen554/goph-keeper/internal/models"
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

	jwt, err := auth.BuildJWTString(userReq.ID)
	if err != nil {
		a.logger.Errorf("cannot build jwt string for authorized user: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	c.SetCookie(auth.CookieName, jwt, maxCookieAge, "", "", false, true)
	res.WriteHeader(http.StatusOK)
}

func (a *App) Logout(c *gin.Context) {
	userID := c.GetUint64(auth.UserIDKey.ToString())
	res := c.Writer
	if userID == 0 {
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	c.SetCookie(auth.CookieName, "", -1, "", "", false, true)
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

	jwt, err := auth.BuildJWTString(userReq.ID)
	if err != nil {
		a.logger.Errorf("cannot build jwt string: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	c.SetCookie(auth.CookieName, jwt, maxCookieAge, "", "", false, true)
	c.JSON(http.StatusOK, userReq)
}

func (a *App) PutDataRecord(c *gin.Context) {
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

	parts := bytes.Split(body, []byte(":"))
	if len(parts) <= 1 {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	data := &models.DataRecord{
		Type:    models.PASS,
		Blocked: false,
	}

	data.Checksum = fmt.Sprintf("%x", md5.Sum(body))
	data.Data = string(body)
	data.UserID = userID

	if err := a.store.PutDataRecord(data, userID); err != nil {
		a.logger.Errorf("unhandled error: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, data)
}

//nolint:dupl // code deduplication will lead to bad code extending in future
func (a *App) GetDataRecords(c *gin.Context) {
	userID := c.GetUint64(auth.UserIDKey.ToString())
	res := c.Writer
	if userID == 0 {
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	orders, err := a.store.GetUserRecords(userID)
	if err != nil {
		if errors.Is(err, models.ErrNoData) {
			res.WriteHeader(http.StatusNoContent)
			return
		}

		a.logger.Errorf("error getting user orders: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, orders)
}

func (a *App) GetDataRecord(c *gin.Context) {
	res := c.Writer
	recordID := c.Param("id")
	preparedRecordID, err := strconv.ParseUint(recordID, 10, 64)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	userID := c.GetUint64(auth.UserIDKey.ToString())

	if userID == 0 {
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	orders, err := a.store.GetUserRecord(preparedRecordID, userID)
	if err != nil {
		if errors.Is(err, models.ErrNoData) {
			res.WriteHeader(http.StatusNoContent)
			return
		}

		a.logger.Errorf("error getting user orders: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, orders)
}

func (a *App) Ping(c *gin.Context) {
	if err := a.store.Ping(); err != nil {
		a.logger.Errorf("Error opening connection to DB: %v", err)
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	c.Writer.WriteHeader(http.StatusOK)
}
