package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/hafiztri123/document-api/config"
	"github.com/hafiztri123/document-api/internal/auth/repository"
	"github.com/hafiztri123/document-api/internal/user/model"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidToken       = errors.New("invalid or expired token")
)

type Service interface {
	Register(ctx context.Context, reg model.UserRegistration) (*model.UserResponse, error)
	Login(ctx context.Context, login model.UserLogin) (*model.TokenResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*model.TokenResponse, error)
	Logout(ctx context.Context, refreshToken string) error
	ValidateToken(tokenString string) (*Claims, error)
}

type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	jwt.RegisteredClaims //Best practice of JWT
}

type authService struct {
	repo repository.Repository
	redis *redis.Client
	logger *zap.Logger
}

func NewAuthService(repo repository.Repository, redis *redis.Client, logger *zap.Logger) Service {
	return &authService{
		repo: repo,
		redis: redis,
		logger: logger,
	}
}

func (s *authService) Register(ctx context.Context, reg model.UserRegistration) (*model.UserResponse, error){
	exisingUser, err := s.repo.FindUserByEmail(ctx, reg.Email)
	if err != nil {
		s.logger.Error("[ERROR] error finding user by email", zap.Error(err))
		return nil, err
	}

	if exisingUser != nil {
		return nil, ErrUserExists
	}

	user := &model.User{
		Email: reg.Email,
		Name: reg.Name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := user.SetPassword(reg.Password); err != nil {
		s.logger.Error("[ERROR] error setting password", zap.Error(err))
		return nil, err
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		s.logger.Error("Error creating user", zap.Error(err))
		return nil, err
	}

	return &model.UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		CreatedAt: user.CreatedAt,
	}, nil
}

func (s *authService) Login(ctx context.Context, login model.UserLogin) (*model.TokenResponse, error){
	user, err := s.repo.FindUserByEmail(ctx, login.Email)
	if err != nil {
		s.logger.Error("[ERROR] error finding user by email", zap.Error(err))
		return nil, err
	}

	//Email is not registered to particular user
	if user == nil {
		return nil, ErrInvalidCredentials
	}


	if !user.CheckPassword(login.Password) {
		return nil, ErrInvalidCredentials
	}

	return s.generateTokens(ctx, user)
}

func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*model.TokenResponse, error){
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(refreshToken, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); ! ok {
			return nil, fmt.Errorf("[ERROR] unexpected signing method: %v", t.Header["alg"])
		}

		return []byte(viper.GetString(config.JWT_SECRET)), nil
	})

	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	//see if refresh token still active in the redis
	key := fmt.Sprintf("refresh_token:%s", refreshToken)
	exists, err := s.redis.Exists(ctx, key).Result()
	if err != nil {
		s.logger.Error("[ERROR] error checking token in redis", zap.Error(err))
		return nil, err
	}
	if exists == 0 {
		return nil, ErrInvalidToken
	}

	user, err := s.repo.FindUserByID(ctx, claims.UserID)
	if err != nil {
		s.logger.Error("[ERROR] error finding user by ID", zap.Error(err))
		return nil, err
	}

	if user == nil {
		return nil, ErrInvalidToken
	}

	// avoid multiple active refresh token
	if err := s.redis.Del(ctx, key).Err(); err != nil {
		s.logger.Error("[ERROR] error deleting fresh token", zap.Error(err))
		return nil, err
	}

	return s.generateTokens(ctx, user)

}

func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(refreshToken, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("[ERROR] unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(viper.GetString(config.JWT_SECRET)), nil
	})

	if err != nil || !token.Valid {
		return ErrInvalidToken
	}

	key := fmt.Sprintf("refresh_token:%s", refreshToken)
	if err := s.redis.Del(ctx, key).Err(); err != nil {
		s.logger.Error("[ERROR] error deleting refresh token", zap.Error(err))
		return err
	}

	return nil
}

func (s *authService) ValidateToken(tokenString string) (*Claims, error){
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("[ERROR] unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(viper.GetString(config.JWT_SECRET)), nil 
	})

	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	//* To fill gin context with claims.userID and claims.email
	return claims, nil

}


func (s *authService) generateTokens(ctx context.Context, user *model.User) (*model.TokenResponse, error) {
	accessExpiryStr := viper.GetString(config.JWT_ACCESS_TOKEN_EXPIRY)
	refreshExpiryStr := viper.GetString(config.JWT_REFRESH_TOKEN_EXPIRY)

	accessExpiry, err := time.ParseDuration(accessExpiryStr)
	if err != nil {
		s.logger.Warn("[WARN] invalid access_token_expiry, using default 15m", zap.Error(err))
		accessExpiry = 15 * time.Minute
	}

	refreshExpiry, err := time.ParseDuration(refreshExpiryStr)
	if err != nil {
		s.logger.Warn("[WARN] invalid refresh_token_expiry, using default 7d", zap.Error(err))
		refreshExpiry = 7 * 24 * time.Hour
	}

	accessClaims := &Claims{
		UserID: user.ID,
		Email: user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(accessExpiry)),
			IssuedAt: jwt.NewNumericDate(time.Now()),
			Subject: user.ID.String(),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(viper.GetString(config.JWT_SECRET)))
	if err != nil {
		s.logger.Error("[ERROR] error signing access token", zap.Error(err))
		return nil, err
	}

	refreshClaims := &Claims{
		UserID: user.ID,
		Email: user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(refreshExpiry)),
			IssuedAt: jwt.NewNumericDate(time.Now()),
			Subject: user.ID.String(),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(viper.GetString(config.JWT_SECRET)))
	if err != nil {
		s.logger.Error("[ERROR] error signing refresh token", zap.Error(err))
		return nil, err
	}

	//to keep track of active refresh token with redis
	key := fmt.Sprintf("refresh_token:%s", refreshTokenString)
	if err := s.redis.Set(ctx, key, user.ID.String(),refreshExpiry).Err(); err != nil {
		s.logger.Error("[ERROR] error storing refresh token in redis", zap.Error(err))
		return nil, err
	}

	return &model.TokenResponse{
		AccessToken: accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresIn: int(accessExpiry.Seconds()),
	}, nil



}