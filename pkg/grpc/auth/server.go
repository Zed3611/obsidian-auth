package authgrpc

import (
	"context"
	"errors"
	"obsidian-auth/pkg/domain/models"
	authservice "obsidian-auth/pkg/service/auth"
	"time"

	authv1 "github.com/Zed3611/obsidian-protos/gen/go/auth/v1"
	validator "github.com/go-playground/validator/v10"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthService interface {
	Register(ctx context.Context, email, password string) (*models.User, error)
	Login(ctx context.Context, email, password, ip, userAgent string) (accessToken, refreshToken string, err error)
	Logout(ctx context.Context, accessToken string) error
	RefreshSession(ctx context.Context, refreshToken string) (newAccessToken, newRefreshToken string, claims *models.Claims, err error)
	GetSessions(ctx context.Context, accessToken string) (sessions *[]models.Session, currentSessionId int, err error)
	RevokeSession(ctx context.Context, accessToken string, sessionId int) error
	RevokeAllSessions(ctx context.Context, accessToken string) (revokedSessionsCount int, err error)
}

type AuthApi struct {
	authv1.UnimplementedAuthServiceServer
	AuthService AuthService
}

func RegisterHandler(server *grpc.Server, authService AuthService) {
	authv1.RegisterAuthServiceServer(server, &AuthApi{AuthService: authService})
}

func (a *AuthApi) Register(ctx context.Context, request *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	v := validator.New()

	if err := v.Var(request.GetEmail(), "required,email"); err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid Email")
	}

	if err := v.Var(request.GetPassword(), "required,min=8"); err != nil {
		return nil, status.Error(codes.InvalidArgument, "Password must be at least 8 symbols")
	}

	user, err := a.AuthService.Register(ctx, request.GetEmail(), request.GetPassword())
	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to register user")
	}

	return &authv1.RegisterResponse{
		UserId: int64(user.Id),
		Email:  user.Email,
	}, nil
}

func (a *AuthApi) Login(ctx context.Context, request *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	v := validator.New()

	if err := v.Var(request.GetEmail(), "required,email"); err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid Email")
	}

	if err := v.Var(request.GetPassword(), "required,min=8"); err != nil {
		return nil, status.Error(codes.InvalidArgument, "Password must be at least 8 symbols")
	}

	accessToken, refreshToken, err := a.AuthService.Login(
		ctx,
		request.GetEmail(),
		request.GetPassword(),
		request.GetIp(),
		request.GetUserAgent(),
	)
	if err != nil {
		if errors.Is(err, authservice.ErrInvalidCredentials) {
			return nil, status.Error(codes.InvalidArgument, "Invalid credentials")
		}

		return nil, status.Error(codes.Internal, "Failed to login")
	}

	return &authv1.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (a *AuthApi) Logout(ctx context.Context, request *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	v := validator.New()

	if err := v.Var(request.GetAccessToken(), "required"); err != nil {
		return nil, status.Error(codes.InvalidArgument, "Token is required")
	}

	if err := a.AuthService.Logout(ctx, request.GetAccessToken()); err != nil {
		if errors.Is(err, authservice.ErrInvalidToken) {
			return nil, status.Error(codes.InvalidArgument, "Invalid access token")
		}

		return nil, status.Error(codes.Internal, "Failed to logout")
	}

	return &authv1.LogoutResponse{}, nil
}

func (a *AuthApi) RefreshToken(ctx context.Context, request *authv1.RefreshTokenRequest) (*authv1.RefreshTokenResponse, error) {
	v := validator.New()

	if err := v.Var(request.GetRefreshToken(), "required"); err != nil {
		return nil, status.Error(codes.InvalidArgument, "Token is required")
	}

	accessToken, refreshToken, _, err := a.AuthService.RefreshSession(ctx, request.GetRefreshToken())
	if err != nil {
		if errors.Is(err, authservice.ErrInvalidToken) {
			return nil, status.Error(codes.InvalidArgument, "Invalid refresh token")
		}

		return nil, status.Error(codes.Internal, "Failed to refresh token")
	}

	return &authv1.RefreshTokenResponse{
		RefreshToken: refreshToken,
		AccessToken:  accessToken,
	}, nil
}

func (a *AuthApi) GetSessions(ctx context.Context, request *authv1.GetSessionsRequest) (*authv1.GetSessionsResponse, error) {
	v := validator.New()

	if err := v.Var(request.GetAccessToken(), "required"); err != nil {
		return nil, status.Error(codes.InvalidArgument, "Token is required")
	}

	sessions, currentSessionId, err := a.AuthService.GetSessions(ctx, request.GetAccessToken())
	if err != nil {
		if errors.Is(err, authservice.ErrInvalidToken) {
			return nil, status.Error(codes.InvalidArgument, "Invalid access token")
		}

		return nil, status.Error(codes.Internal, "Failed to get Sessions")
	}

	_sessions := make([]*authv1.Session, len(*sessions))
	for _, v := range *sessions {
		_sessions = append(_sessions, &authv1.Session{
			Id:        int64(v.Id),
			Ip:        v.Ip,
			UserAgent: v.UserAgent,
			CreatedAt: v.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt: v.UpdatedAt.UTC().Format(time.RFC3339),
			IsCurrent: v.Id == currentSessionId,
		})
	}

	return &authv1.GetSessionsResponse{
		Sessions: _sessions,
	}, nil
}

func (a *AuthApi) RevokeSession(ctx context.Context, request *authv1.RevokeSessionRequest) (*authv1.RevokeSessionResponse, error) {
	v := validator.New()

	if err := v.Var(request.GetAccessToken(), "required"); err != nil {
		return nil, status.Error(codes.InvalidArgument, "Token is required")
	}

	if err := a.AuthService.RevokeSession(ctx, request.GetAccessToken(), int(request.GetSessionId())); err != nil {
		if errors.Is(err, authservice.ErrInvalidToken) {
			return nil, status.Error(codes.InvalidArgument, "Invalid access token")
		}

		return nil, status.Error(codes.Internal, "Failed to revoke session")
	}

	return &authv1.RevokeSessionResponse{}, nil
}

func (a *AuthApi) RevokeAllSessions(ctx context.Context, request *authv1.RevokeAllSessionsRequest) (*authv1.RevokeAllSessionsResponse, error) {
	v := validator.New()

	if err := v.Var(request.GetAccessToken(), "required"); err != nil {
		return nil, status.Error(codes.InvalidArgument, "Token is required")
	}

	count, err := a.AuthService.RevokeAllSessions(ctx, request.GetAccessToken())
	if err != nil {
		if errors.Is(err, authservice.ErrInvalidToken) {
			return nil, status.Error(codes.InvalidArgument, "Invalid access token")
		}

		return nil, status.Error(codes.Internal, "Failed to revoke all sessions")
	}

	return &authv1.RevokeAllSessionsResponse{RevokedCount: int64(count)}, nil
}
