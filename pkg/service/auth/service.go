package authservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"obsidian-auth/pkg/domain/models"
	"obsidian-auth/pkg/lib/log/sl"
	"time"

	"golang.org/x/crypto/bcrypt"

	"obsidian-auth/pkg/storage"
)

type UserRepository interface {
	Create(ctx context.Context, email, passwordHash string) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByID(ctx context.Context, id int) (*models.User, error)
}

type SessionRepository interface {
	Create(ctx context.Context, userId int, ip, userAgent string, activeTil time.Time) (*models.Session, error)
	FindByRefreshTokenHash(ctx context.Context, hash string) (*models.Session, error)
	UpdateRefreshToken(ctx context.Context, sessionID, newHash string, activeTil time.Time) error
	FindByUserID(ctx context.Context, userId string) ([]models.Session, error)
	DeleteOne(ctx context.Context, sessionID, userID string) (bool, error)
	DeleteManyExcept(ctx context.Context, userId, exceptSessionId string) ([]string, error)
}

type Blacklister interface {
	BlacklistSession(sessionId string, duration time.Duration) error
	IsBlacklisted(sessionId string) (bool, error)
}

type AuthService struct {
	u UserRepository
	s SessionRepository
	b Blacklister

	log *slog.Logger

	jwtSecret string
}

var (
	ErrInvalidCredentials = errors.New("Invalid credentials")
)

func (s *AuthService) Register(ctx context.Context, email, password string) (*models.User, error) {
	const op = "service.auth.Register"

	s.log.Info("Attempt to register user")

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		s.log.Error("Failed to generate password hash", sl.Err(err))
		return nil, err
	}

	user, err := s.u.Create(ctx, email, string(hash))
	if err != nil {
		if errors.Is(err, storage.ErrUserExists) { // don't return specific error
			s.log.Debug("User exists", sl.Err(err))
		} else {
			s.log.Error("Failed to register user", sl.Err(err))
		}

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	s.log.Info("User registered successfully")

	return user, nil
}
