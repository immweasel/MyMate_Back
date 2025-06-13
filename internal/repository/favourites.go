package repository

import (
	"context"
	"mymate/pkg/customerror"
	"mymate/pkg/flat"
	"mymate/pkg/user"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FavouritesRepositoryI interface {
	CreateTables(ctx context.Context) error
	GetFavourites(ctx context.Context, offset int64, limit int64, userId uuid.UUID) ([]flat.Flat, error)
	InsertFavourite(ctx context.Context, flat *flat.Flat, user *user.User) (int64, error)
	DeleteFavourite(ctx context.Context, id int64, user *user.User) error
}

type FavouritesRepository struct {
	Pool *pgxpool.Pool
	Host string
	Port string
}

func NewFavouritesRepository(pool *pgxpool.Pool, host string, port string) FavouritesRepositoryI {
	return &FavouritesRepository{
		Pool: pool,
		Host: host,
		Port: port,
	}
}

func (r *FavouritesRepository) CreateTables(ctx context.Context) error {
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS favourites (
		id BIGSERIAL PRIMARY KEY,
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		flat_id BIGINT NOT NULL REFERENCES flat(id) ON DELETE CASCADE,
		CONSTRAINT favourites_user_flat_unique UNIQUE (user_id, flat_id)
	);`
	_, err := r.Pool.Exec(ctx, createTableQuery)
	if err != nil {
		return customerror.NewError("favouritesRepo.CreateTables", r.Host+":"+r.Port, err.Error())
	}
	return nil
}

func (r *FavouritesRepository) GetFavourites(ctx context.Context, offset int64, limit int64, userId uuid.UUID) ([]flat.Flat, error) {
	query := `
		SELECT flat.id, flat.name, flat.about, flat.price_from, flat.price_to, flat.neighborhoods_count, 
		flat.neighborhood_age_from, flat.neighborhood_age_to, flat.sex,
		flat.created_at, flat.created_by_id, flat.up_in_search
		FROM favourites JOIN flat ON favourites.flat_id = flat.id
		WHERE favourites.user_id = $1
		ORDER BY favourites.id DESC LIMIT $2 OFFSET $3; 
	`

	rows, err := r.Pool.Query(ctx, query, userId, limit, offset)
	if err != nil {
		return nil, customerror.NewError("favouritesRepo.GetFavourites", r.Host+":"+r.Port, err.Error())
	}
	var flats []flat.Flat
	for rows.Next() {
		var flat flat.Flat
		err := rows.Scan(&flat.Id, &flat.Name, &flat.About, &flat.PriceFrom, &flat.PriceTo, &flat.NeighborhoodsCount,
			&flat.NeighborhoodAgeFrom, &flat.NeighborhoodAgeTo, &flat.Sex, &flat.CreatedAt, &flat.CreatedById, &flat.UpInSearch)
		if err != nil {
			return nil, customerror.NewError("favouritesRepo.GetFavourites", r.Host+":"+r.Port, err.Error())
		}
		flats = append(flats, flat)
	}
	return flats, nil
}
func (r *FavouritesRepository) InsertFavourite(ctx context.Context, flat *flat.Flat, user *user.User) (int64, error) {
	query := `INSERT INTO favourites (user_id, flat_id) VALUES ($1, $2) RETURNING id`
	var id int64
	err := r.Pool.QueryRow(ctx, query, user.UUID, flat.Id).Scan(&id)
	if err != nil {
		return 0, customerror.NewError("favouritesRepo.InsertFavourite", r.Host+":"+r.Port, err.Error())
	}
	return id, nil
}
func (r *FavouritesRepository) DeleteFavourite(ctx context.Context, flatId int64, user *user.User) error {
	query := `DELETE FROM favourites WHERE flat_id = $1 AND user_id = $2`
	_, err := r.Pool.Exec(ctx, query, flatId, user.UUID)
	if err != nil {
		return customerror.NewError("favouritesRepo.DeleteFavourite", r.Host+":"+r.Port, err.Error())
	}
	return nil
}
