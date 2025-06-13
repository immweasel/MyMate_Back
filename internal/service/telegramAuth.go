package service

import (
	"context"
	"mymate/internal/repository"
	"mymate/pkg/customerror"
	"mymate/pkg/user"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type TelegramAuthenticationServiceI interface {
	SignIn(telegramId int64) (*user.User, error)
	SignUp(telegramId int64, firstname, lastname string) (*user.User, error)
}

type TelegramAuthenticationService struct {
	userRepo repository.UserRepositoryI
	host     string
	port     string
}

func NewTelegramAuthService(userRepo repository.UserRepositoryI, host, port string) TelegramAuthenticationServiceI {
	return &TelegramAuthenticationService{
		userRepo: userRepo,
		host:     host,
		port:     port,
	}
}

func (tAuthService *TelegramAuthenticationService) SignIn(telegramId int64) (*user.User, error) {
	if telegramId == 0 {
		return nil, customerror.ErrWrongCredentials
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	user, err := tAuthService.userRepo.GetUserByCredentials(ctx, "telegram_id", telegramId)
	if err != nil && err != pgx.ErrNoRows {
		customError := err.(customerror.CustomError)
		customError.AppendModule("TelegramAuthenticationService.SignIn")
		return nil, customError
	}
	return user, err
}

func (tAuthService *TelegramAuthenticationService) SignUp(telegramId int64, firstname, lastname string) (*user.User, error) {
	if (firstname == "" && lastname == "") || telegramId == 0 {
		return nil, customerror.ErrWrongCredentials
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	_, err := tAuthService.userRepo.GetUserByCredentials(ctx, "telegram_id", telegramId)
	if err == nil {
		return nil, customerror.ErrUserAlreadyExists
	}
	if err != pgx.ErrNoRows {
		customError := err.(customerror.CustomError)
		customError.AppendModule("TelegramAuthenticationService.SignUp")
		return nil, err
	}
	retries := 0
	for retries < 10 {
		tempUUID, err := uuid.NewRandom()
		if err != nil {
			return nil, customerror.NewError("TelegramAuthenticationService.SignUp.GeneratingUUID", tAuthService.host+":"+tAuthService.port, err.Error())
		}
		var tempUser user.User = user.User{
			UUID:       tempUUID,
			TelegramId: telegramId,
			Firstname:  firstname,
			Lastname:   lastname,
			IsActive:   true,
		}
		err = tAuthService.userRepo.InsertUser(ctx, &tempUser)
		if err == nil {
			return &tempUser, nil
		}
		if err != customerror.ErrUUIDAlreadyExists {
			return nil, customerror.NewError("TelegramAuthenticationService.SignUp.InsertingUser", tAuthService.host+":"+tAuthService.port, err.Error())
		}
		retries += 1
		time.Sleep(2 * time.Second)
	}
	return nil, customerror.NewError("TelegramAuthenticationService.SignUp.InsertingUser", tAuthService.host+":"+tAuthService.port, "Retries Ended")
}
