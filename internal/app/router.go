package app

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/rawen554/goph-keeper/internal/middleware/auth"
	"github.com/rawen554/goph-keeper/internal/middleware/compress"
	ginLogger "github.com/rawen554/goph-keeper/internal/middleware/logger"
)

const (
	rootRoute    = "/"
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
	protectedUserAPI.Use(auth.AuthMiddleware(a.logger))
	{
		protectedUserAPI.POST("/logout", a.Logout)

		recordsAPI := protectedUserAPI.Group("record")
		{
			recordsAPI.POST(rootRoute, a.PutDataRecord)
			recordsAPI.GET(rootRoute, a.GetDataRecords)
			recordsAPI.GET(":id", a.GetDataRecord)
		}
	}

	return r, nil
}
