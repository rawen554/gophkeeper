package auth

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
)

type Claims struct {
	jwt.RegisteredClaims
	UserID uint64
}

type key int

func (k key) ToString() string {
	return fmt.Sprint(k)
}

const (
	tokenExp   = time.Hour * 3
	CookieName = "jwt-token"
)

const UserIDKey key = iota

var ErrTokenNotValid = errors.New("token is not valid")
var ErrNoUserInToken = errors.New("no user data in token")

func BuildJWTString(userID uint64, key string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenExp)),
		},
		UserID: userID,
	})

	tokenString, err := token.SignedString([]byte(key))
	if err != nil {
		return "", fmt.Errorf("error creating signed JWT: %w", err)
	}

	return tokenString, nil
}

func GetUserID(tokenString string, key string) (uint64, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			return []byte(key), nil
		})
	if err != nil {
		if !token.Valid {
			return 0, ErrTokenNotValid
		} else {
			return 0, errors.New("parsing error")
		}
	}

	if claims.UserID == 0 {
		return 0, ErrNoUserInToken
	}

	return claims.UserID, nil
}

func AuthMiddleware(key string, logger *zap.SugaredLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie(CookieName)
		if err != nil {
			logger.Errorf("Error reading cookie[%v]: %v", CookieName, err)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		userID, err := GetUserID(cookie, key)
		if err != nil {
			if errors.Is(err, ErrNoUserInToken) || errors.Is(err, ErrTokenNotValid) {
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			} else {
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
		}

		c.Set(fmt.Sprint(UserIDKey), userID)
		c.Next()
	}
}
