package service

import (
	"context"
	"errors"
	"mymate/internal/repository"
	"mymate/pkg/config"
	"mymate/pkg/customerror"
	"mymate/pkg/user"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Claims struct {
	UserId  uuid.UUID `json:"user_id"`
	Version uint      `json:"version"`
	jwt.RegisteredClaims
}

type JWTServiceI interface {
	GenerateToken(user *user.User, isAccess bool) (string, error)
	ValidateToken(token string) (*user.User, error)
}

type JWTService struct {
	appConfig *config.Config
	userRepo  repository.UserRepositoryI
}

func NewJWTService(appConfig *config.Config, userRepo repository.UserRepositoryI) JWTServiceI {
	return &JWTService{
		appConfig: appConfig,
		userRepo:  userRepo,
	}
}

func (jwtService *JWTService) GenerateToken(user *user.User, isAccess bool) (string, error) {
	expireTime := time.Now().Add(1 * time.Hour)
	if !isAccess {
		expireTime = time.Now().AddDate(0, 1, 0)
	}
	claims := Claims{
		UserId:  user.UUID,
		Version: user.JWTVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireTime),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(jwtService.appConfig.SecretKey))
	if err != nil {
		return "", customerror.NewError("JWTService.GenerateToken", jwtService.appConfig.WebHost+":"+jwtService.appConfig.WebPort, err.Error())
	}

	return tokenString, nil
}

func (jwtService *JWTService) ValidateToken(token string) (*user.User, error) {
	tokenClaims := &Claims{}
	_, err := jwt.ParseWithClaims(token, tokenClaims, func(t *jwt.Token) (interface{}, error) {
		return []byte(jwtService.appConfig.SecretKey), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, err
		}
		return nil, customerror.ErrJwtInvalid
	}
	ctx, close := context.WithTimeout(context.Background(), time.Minute)
	defer close()
	user, err := jwtService.userRepo.GetUser(ctx, tokenClaims.UserId)
	if err == pgx.ErrNoRows {
		return nil, err
	}
	if err != nil {
		customErr := err.(customerror.CustomError)
		customErr.AppendModule("JWTService.ValidateToken")
		return nil, customErr
	}
	if user.JWTVersion != tokenClaims.Version {
		return nil, customerror.ErrJwtVersionIncorrect
	}
	return user, nil

}
