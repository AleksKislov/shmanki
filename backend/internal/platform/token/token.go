package token

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Manager struct {
	secret []byte
	ttl    time.Duration
}

func NewManager(secret string, ttlHours int) *Manager {
	return &Manager{
		secret: []byte(secret),
		ttl:    time.Duration(ttlHours) * time.Hour,
	}
}

func (m *Manager) Generate(userID uuid.UUID) (string, error) {
	now := time.Now().UTC()
	claims := jwt.MapClaims{
		"sub": userID.String(),
		"exp": now.Add(m.ttl).Unix(),
		"iat": now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}

	return signed, nil
}

func (m *Manager) Parse(tokenString string) (uuid.UUID, error) {
	parsedToken, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %s", token.Method.Alg())
		}

		return m.secret, nil
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse jwt: %w", err)
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok || !parsedToken.Valid {
		return uuid.Nil, fmt.Errorf("invalid jwt claims")
	}

	subject, err := claims.GetSubject()
	if err != nil {
		return uuid.Nil, fmt.Errorf("get jwt subject: %w", err)
	}

	userID, err := uuid.Parse(subject)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse user id from jwt: %w", err)
	}

	return userID, nil
}
