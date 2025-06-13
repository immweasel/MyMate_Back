package service

import (
	"context"
	"mymate/internal/repository"
	"mymate/pkg/customerror"
	"mymate/pkg/flat"
	"mymate/pkg/user"
	"time"

	"github.com/google/uuid"
)

type FavouritesServiceI interface {
	GetFavourites(offset int64, limit int64, userId uuid.UUID) ([]flat.Flat, error)
	InsertFavourite(flat *flat.Flat, user *user.User) (int64, error)
	DeleteFavourite(id int64, user *user.User) error
}

type FavouritesService struct {
	favouritesRepo repository.FavouritesRepositoryI
	host           string
	port           string
}

func NewFavouritesService(favouritesRepo repository.FavouritesRepositoryI, host string, port string) FavouritesServiceI {
	return &FavouritesService{
		favouritesRepo: favouritesRepo,
		host:           host,
		port:           port,
	}
}

func (s *FavouritesService) GetFavourites(offset int64, limit int64, userId uuid.UUID) ([]flat.Flat, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	flats, err := s.favouritesRepo.GetFavourites(ctx, offset, limit, userId)
	if err != nil {
		customeErr := err.(customerror.CustomError)
		customeErr.AppendModule("FavouritesService.GetFavourites")
		return []flat.Flat{}, customeErr
	}
	return flats, nil
}

func (s *FavouritesService) InsertFavourite(flat *flat.Flat, user *user.User) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	id, err := s.favouritesRepo.InsertFavourite(ctx, flat, user)
	if err != nil {
		customeErr := err.(customerror.CustomError)
		customeErr.AppendModule("FavouritesService.InsertFavourite")
		return 0, customeErr
	}
	return id, nil
}

func (s *FavouritesService) DeleteFavourite(id int64, user *user.User) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	err := s.favouritesRepo.DeleteFavourite(ctx, id, user)
	if err != nil {
		customeErr := err.(customerror.CustomError)
		customeErr.AppendModule("FavouritesService.DeleteFavourite")
		return customeErr
	}
	return nil
}
