package app

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/rawen554/goph-keeper/internal/middleware/auth"
	"github.com/rawen554/goph-keeper/internal/middleware/compress"
	ginLogger "github.com/rawen554/goph-keeper/internal/middleware/logger"
)

const (
	emptyRoute   = ""
	userAPIRoute = "/api/user"
)

func (a *App) SetupRouter() (*gin.Engine, error) {
	r := gin.New()
	ginLoggerMiddleware, err := ginLogger.Logger(a.logger)
	if err != nil {
		return nil, fmt.Errorf("error creating middleware logger func: %w", err)
	}
	r.Use(ginLoggerMiddleware)
	r.Use(compress.Compress(a.logger))

	r.POST("/api/user/register", a.Register)
	r.POST("/api/user/login", a.Login)

	protectedUserAPI := r.Group(userAPIRoute)
	protectedUserAPI.Use(auth.AuthMiddleware(a.config.Key, a.logger))
	{
		protectedUserAPI.GET("withdrawals", a.GetWithdrawals)
		ordersAPI := protectedUserAPI.Group("orders")
		{
			ordersAPI.POST(emptyRoute, a.PutOrder)
			ordersAPI.GET(emptyRoute, a.GetOrders)
		}

		balanceAPI := protectedUserAPI.Group("balance")
		{
			balanceAPI.GET(emptyRoute, a.GetBalance)
			balanceAPI.POST("withdraw", a.BalanceWithdraw)
		}
	}

	return r, nil
}
