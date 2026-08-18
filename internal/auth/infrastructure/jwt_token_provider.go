package infrastructure

import (
	"fmt"
	"time"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/auth/application"
	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/auth/domain"
	"github.com/golang-jwt/jwt/v5"
)

const defaultTokenExpiry = 24 * time.Hour

type claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type JWTTokenProvider struct {
	key    []byte
	expiry time.Duration
}

func NewJWTTokenProvider(secret string) *JWTTokenProvider {
	return &JWTTokenProvider{key: []byte(secret), expiry: defaultTokenExpiry}
}

var _ application.TokenProvider = (*JWTTokenProvider)(nil)

func (p *JWTTokenProvider) Generate(userID, role string) (string, error) {
	c := claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(p.expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return token.SignedString(p.key)
}

func (p *JWTTokenProvider) Verify(tokenStr string) (userID, role string, err error) {
	token, err := jwt.ParseWithClaims(tokenStr, &claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return p.key, nil
	})
	if err != nil || !token.Valid {
		return "", "", domain.ErrInvalidToken
	}

	c, ok := token.Claims.(*claims)
	if !ok {
		return "", "", domain.ErrInvalidToken
	}
	return c.UserID, c.Role, nil
}
