package service

import (
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"mymate/internal/repository"
	"mymate/pkg/customerror"
	"mymate/pkg/user"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type UserServiceI interface {
	GetUser(id uuid.UUID) (*user.User, error)
	UpdateUser(user *user.User) error
	SaveUserAvatar(user *user.User, file *multipart.FileHeader) error
	DeleteAvatar(user *user.User) error
	DeleteFile(id uuid.UUID, filename string)
}

type UserService struct {
	userRepo repository.UserRepositoryI
	host     string
	port     string
	mainUrl  string
}

func NewUserService(userRepo repository.UserRepositoryI, host string, port string, mainUrl string) UserServiceI {
	return &UserService{
		userRepo: userRepo,
		host:     host,
		port:     port,
		mainUrl:  mainUrl,
	}
}

func (userService *UserService) GetUser(id uuid.UUID) (*user.User, error) {
	ctx, close := context.WithTimeout(context.Background(), time.Minute)
	defer close()
	user, err := userService.userRepo.GetUser(ctx, id)
	if err == pgx.ErrNoRows {
		return nil, err
	}
	if err != nil {
		customErr := err.(customerror.CustomError)
		customErr.AppendModule("UserService.GetUser")
		return nil, customErr
	}
	return user, nil
}

func (userService *UserService) UpdateUser(user *user.User) error {
	ctx, close := context.WithTimeout(context.Background(), time.Minute)
	defer close()
	err := userService.userRepo.UpdateUser(ctx, user)
	if err == pgx.ErrNoRows {
		return err
	}
	if err != nil {
		customErr := err.(customerror.CustomError)
		customErr.AppendModule("UserService.UpdateUser")
		return customErr
	}
	return nil
}

func (userService *UserService) SaveUserAvatar(user *user.User, file *multipart.FileHeader) error {
	tempFilename := user.AvatarFileName
	fileUUID := uuid.New().String()
	timestamp := time.Now().Unix()
	fileExt := filepath.Ext(file.Filename)
	if fileExt != ".jpg" && fileExt != ".jpeg" && fileExt != ".png" && fileExt != ".webp" {
		return customerror.NewError("UserService.SaveUserAvatar.FileExt", userService.host+":"+userService.port, "Invalid file extension")
	}
	uploadPath := filepath.Join(".", "media", user.UUID.String())
	if err := os.MkdirAll(uploadPath, 0755); err != nil {
		return customerror.NewError("UserService.SaveUserAvatar.MkdirAll", userService.host+":"+userService.port, err.Error())
	}
	newFilename := fmt.Sprintf("%s_%d%s", fileUUID, timestamp, fileExt)
	fullPath := filepath.Join(uploadPath, newFilename)
	src, err := file.Open()
	if err != nil {
		return customerror.NewError("UserService.SaveUserAvatar.Open", userService.host+":"+userService.port, err.Error())
	}
	defer src.Close()
	dst, err := os.Create(fullPath)
	if err != nil {
		return customerror.NewError("UserService.SaveUserAvatar.Create", userService.host+":"+userService.port, err.Error())
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	if err != nil {
		return customerror.NewError("UserService.SaveUserAvatar.Copy", userService.host+":"+userService.port, err.Error())
	}
	user.AvatarFileName = newFilename
	user.AvatarUrl = fmt.Sprintf("%s/media/%s/%s", userService.mainUrl, user.UUID.String(), newFilename)
	err = userService.userRepo.UpdateUser(context.Background(), user)
	if err != nil {
		return customerror.NewError("UserService.SaveUserAvatar.UpdateUser", userService.host+":"+userService.port, err.Error())
	}
	if tempFilename != "" {
		go userService.DeleteFile(user.UUID, tempFilename)
	}
	return nil
}
func (userService *UserService) DeleteFile(id uuid.UUID, filename string) {
	err := os.Remove(filepath.Join(".", "media", id.String(), filename))
	if err != nil {
		log.Printf("ERROR|UserService.DeleteFile:%s", err.Error())
		return
	}
}

func (userService *UserService) DeleteAvatar(user *user.User) error {
	tempFilename := user.AvatarFileName
	user.AvatarFileName = ""
	user.AvatarUrl = ""
	ctx, close := context.WithTimeout(context.Background(), time.Minute)
	defer close()
	err := userService.userRepo.UpdateUser(ctx, user)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil
		}
		customErr := err.(customerror.CustomError)
		customErr.AppendModule("UserService.DeleteAvatar")
		return customErr
	}
	if tempFilename != "" {
		go userService.DeleteFile(user.UUID, tempFilename)
	}
	return nil
}
