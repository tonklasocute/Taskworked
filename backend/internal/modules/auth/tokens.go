package auth

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Claims struct {
	UserID string `json:"uid"`
	Role   Role   `json:"role"`
	jwt.RegisteredClaims
}

// TokenService issues/validates JWT access tokens and tracks refresh
// tokens in Redis so "logout everywhere" is a single key delete.
type TokenService struct {
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
	redis         *redis.Client
}

func NewTokenService(accessSecret, refreshSecret string, accessTTL, refreshTTL time.Duration, redis *redis.Client) *TokenService {
	return &TokenService{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
		redis:         redis,
	}
}

func (s *TokenService) IssueAccessToken(u *User) (string, error) {
	claims := Claims{
		UserID: u.ID.String(),
		Role:   u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.accessTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.accessSecret)
}

func (s *TokenService) ParseAccessToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return s.accessSecret, nil
	})
	if err != nil {
		return nil, err
	}
	return claims, nil
}

// IssueRefreshToken creates a random-ish signed token and stores it in
// Redis keyed by user ID (single active refresh token per user).
func (s *TokenService) IssueRefreshToken(ctx context.Context, u *User) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   u.ID.String(),
		ID:        uuid.NewString(),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.refreshTTL)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.refreshSecret)
	if err != nil {
		return "", err
	}
	if err := s.redis.Set(ctx, refreshKey(u.ID.String()), claims.ID, s.refreshTTL).Err(); err != nil {
		return "", err
	}
	return token, nil
}

// ValidateRefreshToken checks the signature and that it matches the token
// currently on record for the user (so a rotated/revoked token fails).
func (s *TokenService) ValidateRefreshToken(ctx context.Context, tokenStr string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return s.refreshSecret, nil
	})
	if err != nil {
		return uuid.Nil, err
	}

	stored, err := s.redis.Get(ctx, refreshKey(claims.Subject)).Result()
	if err != nil || stored != claims.ID {
		return uuid.Nil, jwt.ErrTokenInvalidClaims
	}

	return uuid.Parse(claims.Subject)
}

func (s *TokenService) RevokeRefreshToken(ctx context.Context, userID uuid.UUID) error {
	return s.redis.Del(ctx, refreshKey(userID.String())).Err()
}

func refreshKey(userID string) string {
	return "refresh_token:" + userID
}
