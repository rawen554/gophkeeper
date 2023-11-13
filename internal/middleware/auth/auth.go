package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
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
	tokenExp            = time.Hour * 3
	AuthorizationHeader = "Authorization"
	tokenKey            = "any-key"
)

const UserIDKey key = iota

var ErrTokenNotValid = errors.New("token is not valid")
var ErrNoUserInToken = errors.New("no user data in token")

func BuildJWTString(userID uint64) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenExp)),
		},
		UserID: userID,
	})

	tokenString, err := token.SignedString([]byte(tokenKey))
	if err != nil {
		return "", fmt.Errorf("error creating signed JWT: %w", err)
	}

	return tokenString, nil
}

func GetUserID(tokenString string) (uint64, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			return []byte(tokenKey), nil
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

func AuthMiddleware(logger *zap.SugaredLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader(AuthorizationHeader)
		if token == "" {
			logger.Errorf("Error reading header[%v]: %v", AuthorizationHeader, token)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		splitToken := strings.Split(token, "Bearer ")
		token = splitToken[1]

		userID, err := GetUserID(token)
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
