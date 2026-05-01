package authgrpc

import (
	"context"
	"obsidian-auth/pkg/domain/models"
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

type authApi struct {
	authv1.UnimplementedAuthServiceServer
	authService AuthService
}

func RegisterHandler(server *grpc.Server, authService AuthService) {
	authv1.RegisterAuthServiceServer(server, &authApi{authService: authService})
}

func (a *authApi) Register(ctx context.Context, request *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	v := validator.New()

	if err := v.Var(request.GetEmail(), "required,email"); err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid Email")
	}

	if err := v.Var(request.GetPassword(), "required,min=8"); err != nil {
		return nil, status.Error(codes.InvalidArgument, "Password must be at least 8 symbols")
	}

	user, err := a.authService.Register(ctx, request.GetEmail(), request.GetPassword())
	if err != nil { // TODO error handling
		return nil, status.Error(codes.Internal, "Failed to register user")
	}

	return &authv1.RegisterResponse{
		UserId: int64(user.Id),
		Email:  user.Email,
	}, nil
}

func (a *authApi) Login(ctx context.Context, request *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	v := validator.New()

	if err := v.Var(request.GetEmail(), "required,email"); err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid Email")
	}

	if err := v.Var(request.GetPassword(), "required,min=8"); err != nil {
		return nil, status.Error(codes.InvalidArgument, "Password must be at least 8 symbols")
	}

	accessToken, refreshToken, err := a.authService.Login(ctx, request.GetEmail(), request.GetPassword(), "", "") // TODO add ip and user_agent
	if err != nil {                                                                                               // TODO error handling
		return nil, status.Error(codes.Internal, "Failed to login")
	}

	return &authv1.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (a *authApi) Logout(ctx context.Context, request *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	v := validator.New()

	if err := v.Var(request.GetAccessToken(), "required"); err != nil {
		return nil, status.Error(codes.InvalidArgument, "Token is required")
	}

	if err := a.authService.Logout(ctx, request.GetAccessToken()); err != nil { // TODO error handling
		return nil, status.Error(codes.Internal, "Failed to logout")
	}

	return &authv1.LogoutResponse{}, nil
}

func (a *authApi) RefreshToken(ctx context.Context, request *authv1.RefreshTokenRequest) (*authv1.RefreshTokenResponse, error) {
	v := validator.New()

	if err := v.Var(request.GetRefreshToken(), "required"); err != nil {
		return nil, status.Error(codes.InvalidArgument, "Token is required")
	}

	accessToken, refreshToken, _, err := a.authService.RefreshSession(ctx, request.GetRefreshToken())
	if err != nil { //TODO error handling
		return nil, status.Error(codes.Internal, "Failed to refresh token")
	}

	return &authv1.RefreshTokenResponse{
		RefreshToken: refreshToken,
		AccessToken:  accessToken,
	}, nil
}

func (a *authApi) GetSessions(ctx context.Context, request *authv1.GetSessionsRequest) (*authv1.GetSessionsResponse, error) {
	v := validator.New()

	if err := v.Var(request.GetAccessToken(), "required"); err != nil {
		return nil, status.Error(codes.InvalidArgument, "Token is required")
	}

	sessions, currentSessionId, err := a.authService.GetSessions(ctx, request.GetAccessToken())
	if err != nil { //TODO error handling
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

func (a *authApi) RevokeSession(ctx context.Context, request *authv1.RevokeSessionRequest) (*authv1.RevokeSessionResponse, error) {
	v := validator.New()

	if err := v.Var(request.GetAccessToken(), "required"); err != nil {
		return nil, status.Error(codes.InvalidArgument, "Token is required")
	}

	if err := a.authService.RevokeSession(ctx, request.GetAccessToken(), int(request.GetSessionId())); err != nil { //TODO error handling
		return nil, status.Error(codes.Internal, "Failed to revoke session")
	}

	return &authv1.RevokeSessionResponse{}, nil
}

func (a *authApi) RevokeAllSessions(ctx context.Context, request *authv1.RevokeAllSessionsRequest) (*authv1.RevokeAllSessionsResponse, error) {
	v := validator.New()

	if err := v.Var(request.GetAccessToken(), "required"); err != nil {
		return nil, status.Error(codes.InvalidArgument, "Token is required")
	}

	count, err := a.authService.RevokeAllSessions(ctx, request.GetAccessToken())
	if err != nil { //TODO error handling
		return nil, status.Error(codes.Internal, "Failed to revoke all sessions")
	}

	return &authv1.RevokeAllSessionsResponse{RevokedCount: int64(count)}, nil
}
