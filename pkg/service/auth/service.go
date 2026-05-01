package authservice

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"obsidian-auth/pkg/domain/models"
	"obsidian-auth/pkg/lib/log/sl"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"obsidian-auth/pkg/storage"
)

type UserRepository interface {
	Create(ctx context.Context, email, passwordHash string) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByID(ctx context.Context, id int) (*models.User, error)
}

type SessionRepository interface {
	Create(ctx context.Context, userId int, ip, userAgent, tokenHash string, activeTil time.Time) (*models.Session, error)
	FindByRefreshTokenHash(ctx context.Context, hash string) (*models.Session, error)
	UpdateRefreshToken(ctx context.Context, sessionId int, newHash string, activeTil time.Time) error
	FindByUserID(ctx context.Context, userId int) (*[]models.Session, error)
	DeleteOne(ctx context.Context, sessionId, userId int) error
	DeleteManyExcept(ctx context.Context, userId, exceptSessionId int) ([]int, error)
}

type Blacklister interface {
	BlacklistSession(ctx context.Context, sessionId int, duration time.Duration) error
	IsBlacklisted(ctx context.Context, sessionId int) (bool, error)
}

type AuthService struct {
	u UserRepository
	s SessionRepository
	b Blacklister

	log *slog.Logger

	cfg AuthConfig
}

type AuthConfig struct {
	JwtSecret           string
	SessionDuration     time.Duration
	AccessTokenDuration time.Duration
}

var (
	ErrInvalidToken            = errors.New("Invalid token")
	ErrInvalidCredentials      = errors.New("Invalid credentials")
	ErrUnexpectedSigningMethod = errors.New("Unexpected signing method of token")
)

func New(cfg AuthConfig, log *slog.Logger, b Blacklister, s SessionRepository, u UserRepository) *AuthService {
	return &AuthService{
		u:   u,
		s:   s,
		b:   b,
		log: log,
		cfg: cfg,
	}
}

func (a *AuthService) Register(ctx context.Context, email, password string) (*models.User, error) {
	const op = "service.auth.Register"

	a.log.Info("Attempt to register user")

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		a.log.Error("Failed to generate password hash", sl.Err(err))
		return nil, err
	}

	user, err := a.u.Create(ctx, email, string(hash))
	if err != nil {
		if errors.Is(err, storage.ErrUserExists) { // don't return specific error
			a.log.Debug("User exists", sl.Err(err))
		} else {
			a.log.Error("Failed to register user", sl.Err(err))
		}

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	a.log.Info("User registered successfully")

	return user, nil
}

func (a *AuthService) Login(ctx context.Context, email, password, ip, userAgent string) (string, string, error) {
	const op = "service.auth.Login"

	a.log.Info("Attempt to login user")

	user, err := a.u.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			a.log.Debug("User not found", sl.Err(err))

			return "", "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
		}

		a.log.Error("Failed to login user", sl.Err(err))

		return "", "", fmt.Errorf("%s: %w", op, err)
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		a.log.Debug("Password don't match", sl.Err(err))
		return "", "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
	}

	refreshToken, err := generateRefreshToken()
	if err != nil {
		a.log.Error("Failed to generate new refresh token", sl.Err(err))

		return "", "", fmt.Errorf("%s: %w", op, err)
	}

	hash := hashToken(refreshToken)

	session, err := a.s.Create(ctx, user.Id, ip, userAgent, hash, time.Now().Add(a.cfg.SessionDuration))
	if err != nil {
		a.log.Error("Failed to create new user session", sl.Err(err))

		return "", "", fmt.Errorf("%s: %w", op, err)
	}

	accessToken, _, err := a.generateAccessToken(user.Id, user.Email, session.Id)
	if err != nil {
		a.log.Error("Failed to generate new access token", sl.Err(err))

		return "", "", fmt.Errorf("%s: %w", op, err)
	}

	a.log.Info("User logged in successfully")

	return accessToken, refreshToken, nil
}

func (a *AuthService) RefreshSession(ctx context.Context, refreshToken string) (string, string, *models.Claims, error) {
	const op = "service.auth.RefreshSession"

	a.log.Info("Attempt to refresh session")

	hash := hashToken(refreshToken)
	session, err := a.s.FindByRefreshTokenHash(ctx, hash)
	if err != nil {
		if errors.Is(err, storage.ErrSessionNotFound) {
			a.log.Debug("Session not found", sl.Err(err))
			return "", "", nil, fmt.Errorf("%s: %w", op, ErrInvalidToken)
		}

		a.log.Error("Failed to find session", sl.Err(err))

		return "", "", nil, fmt.Errorf("%s: %w", op, err)
	}

	user, err := a.u.FindByID(ctx, session.UserId)
	if err != nil {
		a.log.Error("Failed to find session", sl.Err(err))

		return "", "", nil, fmt.Errorf("%s: %w", op, err)
	}

	newRefreshToken, err := generateRefreshToken()
	if err != nil {
		a.log.Error("Failed to generate new refresh token", sl.Err(err))

		return "", "", nil, fmt.Errorf("%s: %w", op, err)
	}

	newHash := hashToken(newRefreshToken)
	err = a.s.UpdateRefreshToken(ctx, session.Id, newHash, time.Now().Add(a.cfg.SessionDuration))
	if err != nil {
		a.log.Error("Failed to update refresh session", sl.Err(err))

		return "", "", nil, fmt.Errorf("%s: %w", op, err)
	}

	accessToken, claims, err := a.generateAccessToken(user.Id, user.Email, session.Id)
	if err != nil {
		a.log.Error("Failed to generate access token", sl.Err(err))

		return "", "", nil, fmt.Errorf("%s: %w", op, err)
	}

	a.log.Info("Session refreshed successfully")

	return accessToken, refreshToken, claims, nil
}

func (a *AuthService) generateAccessToken(userId int, email string, sessionId int) (string, *models.Claims, error) {
	const op = "service.auth.generateAccessToken"

	now := time.Now()
	expireAt := now.Add(a.cfg.AccessTokenDuration)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   userId,
		"email": email,
		"sid":   sessionId,
		"iat":   now.Unix(),
		"exp":   expireAt.Unix(),
	})
	tokenStr, err := token.SignedString(a.cfg.JwtSecret)
	if err != nil {
		return tokenStr, nil, fmt.Errorf("%s: %w", op, err)
	}
	claims := &models.Claims{
		UserId:         userId,
		Email:          email,
		SessionId:      sessionId,
		ExpirationTime: expireAt,
	}
	return tokenStr, claims, nil
}

func (a *AuthService) validateAccessToken(tokenStr string) (*models.Claims, error) {
	const op = "service.auth.validateAccessToken"

	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrUnexpectedSigningMethod
		}

		return a.cfg.JwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	sub, _ := claims["sub"].(int)
	email, _ := claims["email"].(string)
	sid, _ := claims["sid"].(int)
	et, _ := claims.GetExpirationTime()

	return &models.Claims{UserId: sub, Email: email, SessionId: sid, ExpirationTime: et.Time}, nil
}

func (a *AuthService) GetSessions(ctx context.Context, accessToken string) (*[]models.Session, int, error) {
	const op = "service.auth.GetSessions"

	a.log.Info("Attempt to get sessions")

	claims, err := a.validateAccessToken(accessToken)
	if err != nil {
		a.log.Debug("Error validating access token", sl.Err(err))

		return nil, 0, fmt.Errorf("%s: %w", op, ErrInvalidToken)
	}

	sessions, err := a.s.FindByUserID(ctx, claims.UserId)
	if err != nil {
		a.log.Debug("Failed to get user sessions by user id", sl.Err(err))

		return nil, 0, fmt.Errorf("%s: %w", op, err)
	}

	a.log.Info("Sessions returned successfully")

	return sessions, claims.SessionId, nil
}

func (a *AuthService) RevokeSession(ctx context.Context, accessToken string, sessionId int) error {
	const op = "service.auth.RevokeSession"

	a.log.Info("Attempt to revoke session")

	claims, err := a.validateAccessToken(accessToken)
	if err != nil {
		a.log.Debug("Error validating access token", sl.Err(err))

		return fmt.Errorf("%s: %w", op, ErrInvalidToken)
	}

	err = a.s.DeleteOne(ctx, sessionId, claims.UserId)
	if err != nil {
		a.log.Debug("Error removing session", sl.Err(err))

		return fmt.Errorf("%s: %w", op, err)
	}

	if err := a.b.BlacklistSession(ctx, sessionId, a.cfg.AccessTokenDuration); err != nil {
		a.log.Debug("Error blacklisting session", sl.Err(err))

		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (a *AuthService) RevokeAllSessions(ctx context.Context, accessToken string) (revokedSessionsCount int, err error) {
	const op = "service.auth.RevokeAllSessions"

	a.log.Info("Attempt to revoke all sessions")

	claims, err := a.validateAccessToken(accessToken)
	if err != nil {
		a.log.Debug("Error validating access token", sl.Err(err))

		return 0, fmt.Errorf("%s: %w", op, ErrInvalidToken)
	}

	deleted, err := a.s.DeleteManyExcept(ctx, claims.UserId, claims.SessionId)
	if err != nil {
		a.log.Debug("Error removing sessions", sl.Err(err))

		return 0, fmt.Errorf("%s: %w", op, err)
	}

	for _, id := range deleted {
		_ = a.b.BlacklistSession(ctx, id, a.cfg.AccessTokenDuration)
	}

	a.log.Info("All sessions revoked")

	return len(deleted), nil
}

func (a *AuthService) Logout(ctx context.Context, accessToken string) error {
	const op = "service.auth.Logout"

	a.log.Info("Attempt to logout")

	claims, err := a.validateAccessToken(accessToken)
	if err != nil {
		a.log.Debug("Error validating access token", sl.Err(err))

		return fmt.Errorf("%s: %w", op, ErrInvalidToken)
	}

	err = a.s.DeleteOne(ctx, claims.SessionId, claims.UserId)
	if err != nil {
		a.log.Debug("Error removing session", sl.Err(err))

		return fmt.Errorf("%s: %w", op, err)
	}

	if err := a.b.BlacklistSession(ctx, claims.SessionId, a.cfg.AccessTokenDuration); err != nil {
		a.log.Debug("Error blacklisting session", sl.Err(err))

		return fmt.Errorf("%s: %w", op, err)
	}

	a.log.Info("User logged out successfully")

	return nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func generateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
