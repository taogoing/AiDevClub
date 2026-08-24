package platform

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type accessClaims struct {
	UserID uint `json:"uid"`
	jwt.RegisteredClaims
}

func GenerateAccessToken(secret string, ttl time.Duration, userID uint) (string, error) {
	now := time.Now()
	claims := accessClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

var ErrInvalidToken = errors.New("invalid token")

func ParseAccessToken(secret, token string) (uint, error) {
	t, err := jwt.ParseWithClaims(token, &accessClaims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return 0, ErrInvalidToken
	}
	claims, ok := t.Claims.(*accessClaims)
	if !ok || !t.Valid {
		return 0, ErrInvalidToken
	}
	return claims.UserID, nil
}
