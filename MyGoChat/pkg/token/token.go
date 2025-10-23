package token

import (
	"MyGoChat/internal/model"
	"MyGoChat/pkg/config"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserUuid string `json:"useruuid"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// GenerateToken 为给定的用户生成JWT令牌。
func GenerateToken(user *model.User) (string, error) {
	jwtSecret := config.GetConfig().JwtSecret.SecretKey
	if jwtSecret == "" {
		return "", errors.New("JWT secret not configured")
	}

	expirationTime := time.Now().Add(24 * time.Hour)

	claims := &Claims{
		UserUuid: user.Uuid,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtSecret))

	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ParseToken 解析并验证JWT令牌。
func ParseToken(tokenString string) (*Claims, error) {
	jwtSecret := config.GetConfig().JwtSecret.SecretKey
	if jwtSecret == "" {
		return nil, errors.New("JWT secret not configured")
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

func GetUserUuidFromToken(tokenString string) (string, error) {
	claims, err := ParseToken(tokenString)
	if err != nil {
		return "", err
	}
	return claims.UserUuid, nil
}
