package service

import (
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"mymate/internal/repository"
	"mymate/pkg/customerror"
	modelsFlat "mymate/pkg/flat"
	"mymate/pkg/user"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type FlatServiceI interface {
	GetFlats(offset int64, limit int64, filters map[string]any) ([]modelsFlat.Flat, error)
	GetFlat(id int64) (*modelsFlat.Flat, error)
	InsertFlat(flat *modelsFlat.Flat) (int64, error)
	UpdateFlat(flat *modelsFlat.Flat, user *user.User) error
	DeleteFlat(id int64, user *user.User) error
	GetFlatImages(flatId int64) ([]modelsFlat.FlatImage, error)
	InsertFlatImage(file *multipart.FileHeader, user *modelsFlat.Flat) error
	DeleteFlatImage(flatImage *modelsFlat.FlatImage) error
}

type FlatService struct {
	flatRepo repository.FlatRepositoryI
	host     string
	port     string
	mainUrl  string
}

func NewFlatService(flatRepo repository.FlatRepositoryI, host string, port string, mainUrl string) FlatServiceI {
	return &FlatService{
		flatRepo: flatRepo,
		host:     host,
		port:     port,
		mainUrl:  mainUrl,
	}
}

func (flatService *FlatService) GetFlats(offset int64, limit int64, filters map[string]any) ([]modelsFlat.Flat, error) {
	ctx, close := context.WithTimeout(context.Background(), time.Minute)
	defer close()
	flats, err := flatService.flatRepo.GetFlats(ctx, offset, limit, filters)
	if err != nil {
		customeErr := err.(customerror.CustomError)
		customeErr.AppendModule("FlatService.GetFlats")
		return []modelsFlat.Flat{}, customeErr
	}
	return flats, nil
}

func (flatService *FlatService) GetFlat(id int64) (*modelsFlat.Flat, error) {
	ctx, close := context.WithTimeout(context.Background(), time.Minute)
	defer close()
	flat, err := flatService.flatRepo.GetFlat(ctx, id)
	if err != nil {
		customeErr := err.(customerror.CustomError)
		customeErr.AppendModule("FlatService.GetFlat")
		return nil, customeErr
	}
	return flat, nil
}

func (flatService *FlatService) InsertFlat(flat *modelsFlat.Flat) (int64, error) {
	ctx, close := context.WithTimeout(context.Background(), time.Minute)
	defer close()
	id, err := flatService.flatRepo.InsertFlat(ctx, flat)
	if err != nil {
		customeErr := err.(customerror.CustomError)
		customeErr.AppendModule("FlatService.InsertFlat")
		return id, customeErr
	}
	return id, nil
}

func (flatService *FlatService) UpdateFlat(flat *modelsFlat.Flat, user *user.User) error {
	ctx, close := context.WithTimeout(context.Background(), time.Minute)
	defer close()
	err := flatService.flatRepo.UpdateFlat(ctx, flat, user)
	if err == pgx.ErrNoRows {
		return err
	}
	if err != nil {
		customeErr := err.(customerror.CustomError)
		customeErr.AppendModule("FlatService.UpdateFlat")
		return customeErr
	}
	return nil
}

func (flatService *FlatService) DeleteFlat(id int64, user *user.User) error {
	ctx, close := context.WithTimeout(context.Background(), time.Minute)
	defer close()
	err := flatService.flatRepo.DeleteFlat(ctx, id, user)
	if err == pgx.ErrNoRows {
		return err
	}
	if err != nil {
		customeErr := err.(customerror.CustomError)
		customeErr.AppendModule("FlatService.DeleteFlat")
		return customeErr
	}
	return nil
}

func (flatService *FlatService) GetFlatImages(flatId int64) ([]modelsFlat.FlatImage, error) {
	ctx, close := context.WithTimeout(context.Background(), time.Minute)
	defer close()
	flatImages, err := flatService.flatRepo.GetFlatImages(ctx, flatId)
	if err != nil {
		customeErr := err.(customerror.CustomError)
		customeErr.AppendModule("FlatService.GetFlatImages")
		return []modelsFlat.FlatImage{}, customeErr
	}
	return flatImages, nil
}
func (flatService *FlatService) InsertFlatImage(file *multipart.FileHeader, flat *modelsFlat.Flat) error {
	fileUUID := uuid.New().String()
	timestamp := time.Now().Unix()
	fileExt := filepath.Ext(file.Filename)
	uploadPath := filepath.Join(".", "media", "flats", strconv.FormatInt(flat.Id, 10))
	if err := os.MkdirAll(uploadPath, 0755); err != nil {
		return customerror.NewError("FlatService.SaveUserAvatar.MkdirAll", flatService.host+":"+flatService.port, err.Error())
	}
	if fileExt != ".jpg" && fileExt != ".jpeg" && fileExt != ".png" && fileExt != ".webp" {
		return customerror.NewError("FlatService.SaveUserAvatar.FileExt", flatService.host+":"+flatService.port, "Invalid file extension")
	}
	newFilename := fmt.Sprintf("%s_%d%s", fileUUID, timestamp, fileExt)
	fullPath := filepath.Join(uploadPath, newFilename)
	src, err := file.Open()
	if err != nil {
		return customerror.NewError("FlatService.SaveUserAvatar.Open", flatService.host+":"+flatService.port, err.Error())
	}
	defer src.Close()
	dst, err := os.Create(fullPath)
	if err != nil {
		return customerror.NewError("FlatService.SaveUserAvatar.Create", flatService.host+":"+flatService.port, err.Error())
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	if err != nil {
		return customerror.NewError("FlatService.SaveUserAvatar.Copy", flatService.host+":"+flatService.port, err.Error())
	}
	flatImage := modelsFlat.FlatImage{
		FlatId:   flat.Id,
		Url:      fmt.Sprintf("%s/media/flats/%d/%s", flatService.mainUrl, flat.Id, newFilename),
		Filename: newFilename,
	}
	c, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	err = flatService.flatRepo.InsertFlatImage(c, &flatImage)
	if err != nil {
		return customerror.NewError("FlatService.InsertFlatImage", flatService.host+":"+flatService.port, err.Error())
	}
	return nil
}

func (flatService *FlatService) DeleteFlatImage(flatImage *modelsFlat.FlatImage) error {
	tempFilename := flatImage.Filename
	tempFlatId := flatImage.FlatId
	ctx, close := context.WithTimeout(context.Background(), time.Minute)
	defer close()
	err := flatService.flatRepo.DeleteFlatImage(ctx, flatImage)
	if err == pgx.ErrNoRows {
		return err
	}
	if err != nil {
		customeErr := err.(customerror.CustomError)
		customeErr.AppendModule("FlatService.DeleteFlatImage")
		return customeErr
	}
	go flatService.DeleteFile(filepath.Join(".", "media", "flats", strconv.FormatInt(tempFlatId, 10), tempFilename))
	return nil
}

func (flatService *FlatService) DeleteFile(path string) {
	err := os.Remove(path)
	fmt.Println(path)
	if err != nil {
		customeErr := customerror.NewError("FlatService.DeleteFile", flatService.host+":"+flatService.port, err.Error()).(customerror.CustomError)
		customeErr.AppendModule("FlatService.DeleteFile")
		log.Println(customeErr)
		return
	}
}
