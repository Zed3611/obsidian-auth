package authgrpc_test

import (
	"context"
	"errors"
	"fmt"
	"obsidian-auth/pkg/domain/models"
	authgrpc "obsidian-auth/pkg/grpc/auth"
	authservice "obsidian-auth/pkg/service/auth"
	"testing"

	obsidian_auth_v1 "github.com/Zed3611/obsidian-protos/gen/go/auth/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthServiceMock struct {
	mock.Mock
}

func (m *AuthServiceMock) Register(ctx context.Context, email, password string) (*models.User, error) {
	args := m.Called(ctx, email, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}
func (m *AuthServiceMock) Login(ctx context.Context, email, password, ip, userAgent string) (string, string, error) {
	args := m.Called(ctx, email, password, ip, userAgent)
	return args.String(0), args.String(1), args.Error(2)
}
func (m *AuthServiceMock) Logout(ctx context.Context, accessToken string) error {
	args := m.Called(ctx, accessToken)
	return args.Error(0)
}
func (m *AuthServiceMock) RefreshSession(ctx context.Context, refreshToken string) (string, string, *models.Claims, error) {
	args := m.Called(ctx, refreshToken)
	return args.String(0), args.String(1), args.Get(2).(*models.Claims), args.Error(3)
}
func (m *AuthServiceMock) GetSessions(ctx context.Context, accessToken string) (*[]models.Session, int, error) {
	args := m.Called(ctx, accessToken)
	return args.Get(0).(*[]models.Session), args.Int(1), args.Error(2)
}
func (m *AuthServiceMock) RevokeSession(ctx context.Context, accessToken string, sessionId int) error {
	args := m.Called(ctx, accessToken, sessionId)
	return args.Error(0)
}
func (m *AuthServiceMock) RevokeAllSessions(ctx context.Context, accessToken string) (revokedSessionsCount int, err error) {
	args := m.Called(ctx, accessToken)
	return args.Int(0), args.Error(1)
}

func provideAuthApi() (*authgrpc.AuthApi, *AuthServiceMock) {
	mock := new(AuthServiceMock)
	return &authgrpc.AuthApi{
		AuthService: mock,
	}, mock
}

func TestRegister_Success(t *testing.T) {
	api, authMock := provideAuthApi()
	ctx := context.Background()

	const password = "password"
	testUser := &models.User{
		Id:    321,
		Email: "test@example.com",
	}
	authMock.On("Register", mock.Anything, testUser.Email, password).Return(testUser, nil)

	res, err := api.Register(ctx, &obsidian_auth_v1.RegisterRequest{
		Email:    testUser.Email,
		Password: password,
	})

	assert.NoError(t, err)
	assert.Equal(t, int64(testUser.Id), res.GetUserId())
	assert.Equal(t, testUser.Email, res.GetEmail())
}

func TestRegister_UnhandledError(t *testing.T) {
	api, authMock := provideAuthApi()
	ctx := context.Background()

	const password = "password"
	const email = "test@example.com"
	authMock.On("Register", mock.Anything, email, password).Return(nil, errors.New("Unhandled error"))

	_, err := api.Register(ctx, &obsidian_auth_v1.RegisterRequest{
		Email:    email,
		Password: password,
	})

	assert.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestRegister_Validation(t *testing.T) {
	api, authMock := provideAuthApi()
	ctx := context.Background()

	type validationTestCase struct {
		email     string
		password  string
		isError   bool
		errorCode codes.Code
	}

	testCases := []validationTestCase{
		{"test", "12345678", true, codes.InvalidArgument},
		{"", "12345678", true, codes.InvalidArgument},
		{"test@example.com", "1234567", true, codes.InvalidArgument},
		{"test@example.com", "", true, codes.InvalidArgument},
		{email: "test@example.com", password: "12345678", isError: false},
	}

	for _, testCase := range testCases {
		if !testCase.isError {
			authMock.On("Register", mock.Anything, testCase.email, testCase.password).Return(&models.User{}, nil)
		}

		_, err := api.Register(ctx, &obsidian_auth_v1.RegisterRequest{
			Email:    testCase.email,
			Password: testCase.password,
		})

		if testCase.isError {
			assert.Error(t, err)
			assert.Equal(t, testCase.errorCode, status.Code(err))
		} else {
			assert.NoError(t, err)
		}
	}
}

func TestLogin_Success(t *testing.T) {
	api, authMock := provideAuthApi()
	ctx := context.Background()

	const password = "password"
	const email = "test@example.com"
	const accessToken = "test-access"
	const refreshToken = "test-refresh"

	authMock.
		On("Login", mock.Anything, email, password, "", "").
		Return(accessToken, refreshToken, nil)

	res, err := api.Login(ctx, &obsidian_auth_v1.LoginRequest{Email: email, Password: password})

	assert.NoError(t, err)
	assert.Equal(t, accessToken, res.GetAccessToken())
	assert.Equal(t, refreshToken, res.GetRefreshToken())
}

func TestLogin_Validation(t *testing.T) {
	api, authMock := provideAuthApi()
	ctx := context.Background()

	type validationTestCase struct {
		email     string
		password  string
		isError   bool
		errorCode codes.Code
	}

	testCases := []validationTestCase{
		{"test", "12345678", true, codes.InvalidArgument},
		{"", "12345678", true, codes.InvalidArgument},
		{"test@example.com", "1234567", true, codes.InvalidArgument},
		{"test@example.com", "", true, codes.InvalidArgument},
		{email: "test@example.com", password: "12345678", isError: false},
	}

	for _, testCase := range testCases {
		if !testCase.isError {
			authMock.On("Login", mock.Anything, testCase.email, testCase.password, "", "").Return("", "", nil)
		}

		_, err := api.Login(ctx, &obsidian_auth_v1.LoginRequest{
			Email:    testCase.email,
			Password: testCase.password,
		})

		if testCase.isError {
			assert.Error(t, err)
			assert.Equal(t, testCase.errorCode, status.Code(err))
		} else {
			assert.NoError(t, err)
		}
	}
}

func TestLogin_InvalidCreds(t *testing.T) {
	api, authMock := provideAuthApi()
	ctx := context.Background()

	const password = "password"
	const email = "test@example.com"
	authMock.
		On("Login", mock.Anything, email, password, "", "").
		Return("", "", fmt.Errorf("%s: %w", "TestLogin_InvalidCreds", authservice.ErrInvalidCredentials))

	_, err := api.Login(ctx, &obsidian_auth_v1.LoginRequest{Email: email, Password: password})

	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

// ...
