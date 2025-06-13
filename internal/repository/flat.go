package repository

import (
	"context"
	"errors"
	"fmt"
	"mymate/pkg/customerror"
	"mymate/pkg/flat"
	"mymate/pkg/user"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FlatRepositoryI interface {
	CreateTables(ctx context.Context) error
	GetFlats(ctx context.Context, offset int64, limit int64, filters map[string]any) ([]flat.Flat, error)
	GetFlat(ctx context.Context, id int64) (*flat.Flat, error)
	InsertFlat(ctx context.Context, flat *flat.Flat) (int64, error)
	UpdateFlat(ctx context.Context, flat *flat.Flat, user *user.User) error
	DeleteFlat(ctx context.Context, id int64, user *user.User) error

	GetFlatImages(ctx context.Context, flatId int64) ([]flat.FlatImage, error)
	InsertFlatImage(ctx context.Context, flatImage *flat.FlatImage) error
	DeleteFlatImage(ctx context.Context, flatImage *flat.FlatImage) error
}

type FlatRepository struct {
	Pool    *pgxpool.Pool
	Host    string
	Port    string
	WebName string
}

func NewFlatRepository(pool *pgxpool.Pool, host string, port string, webname string) FlatRepositoryI {
	return &FlatRepository{
		Pool:    pool,
		Host:    host,
		Port:    port,
		WebName: webname,
	}
}

func (flatRepo *FlatRepository) CreateTables(ctx context.Context) error {
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS flat (
		id BIGSERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		about TEXT NOT NULL,
		price_from BIGINT DEFAULT 0,
		price_to BIGINT DEFAULT 0,
		neighborhoods_count INTEGER DEFAULT 0,
		neighborhood_age_from INTEGER DEFAULT 0,
		neighborhood_age_to INTEGER DEFAULT 0,
		sex TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		created_by_id UUID NOT NULL REFERENCES users(id),
		up_in_search INTEGER DEFAULT 0
	);`
	_, err := flatRepo.Pool.Exec(ctx, createTableQuery)
	if err != nil {
		return customerror.NewError("flatRepo.CreateTables", flatRepo.Host+":"+flatRepo.Port, err.Error())
	}

	createFlatImageQuery := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS flat_image (
		id BIGSERIAL PRIMARY KEY,
		flat_id BIGINT NOT NULL REFERENCES flat(id) ON DELETE CASCADE,
		url TEXT NOT NULL DEFAULT '%s/media/flat/placeholder.png',
		filename TEXT NOT NULL DEFAULT ''
	);`, flatRepo.WebName)

	_, err = flatRepo.Pool.Exec(ctx, createFlatImageQuery)
	if err != nil {
		return customerror.NewError("flatRepo.CreateTables", flatRepo.Host+":"+flatRepo.Port, err.Error())
	}

	createIndexQuery := `CREATE INDEX IF NOT EXISTS flat_id_idx ON flat(id);`
	_, err = flatRepo.Pool.Exec(ctx, createIndexQuery)
	if err != nil {
		return customerror.NewError("flatRepo.CreateTables", flatRepo.Host+":"+flatRepo.Port, err.Error())
	}

	createIndexQuery = `CREATE INDEX IF NOT EXISTS flat_created_by_id_idx ON flat(created_by_id);`
	_, err = flatRepo.Pool.Exec(ctx, createIndexQuery)
	if err != nil {
		return customerror.NewError("flatRepo.CreateTables", flatRepo.Host+":"+flatRepo.Port, err.Error())
	}

	createIndexQuery = `CREATE INDEX IF NOT EXISTS flat_image_idx ON flat_image(flat_id);`
	_, err = flatRepo.Pool.Exec(ctx, createIndexQuery)
	if err != nil {
		return customerror.NewError("flatRepo.CreateTables", flatRepo.Host+":"+flatRepo.Port, err.Error())
	}
	return nil
}

func (flatRepo *FlatRepository) GetFlats(ctx context.Context, offset int64, limit int64, filters map[string]any) ([]flat.Flat, error) {
	flats := []flat.Flat{}
	filtersCount := 1
	query := `SELECT flat.id, flat.name, flat.about, flat.price_from, flat.price_to, flat.neighborhoods_count, 
	flat.neighborhood_age_from, flat.neighborhood_age_to, flat.sex,
	flat.created_at, flat.created_by_id, flat.up_in_search, users.id, users.firstname, users.lastname, users.avatar_url 
	FROM flat JOIN users ON flat.created_by_id = users.id WHERE flat.id IS NOT NULL`
	params := []any{}
	fmt.Print(filters)
	if filters["name"] != nil {
		query += " AND strpos(flat.name, $" + fmt.Sprint(filtersCount) + ") > 0 "
		params = append(params, filters["name"])
		filtersCount++
	}

	if filters["about"] != nil {
		query += " AND strpos(flat.about, $" + fmt.Sprint(filtersCount) + ") > 0 "
		params = append(params, filters["about"])
		filtersCount++
	}

	if filters["price_from"] != nil {
		query += " AND flat.price_from >= $" + fmt.Sprint(filtersCount)
		params = append(params, filters["price_from"])
		filtersCount++
	}

	if filters["price_to"] != nil {
		query += " AND flat.price_to <= $" + fmt.Sprint(filtersCount)
		params = append(params, filters["price_to"])
		filtersCount++
	}

	if filters["neighborhoods_count_from"] != nil {
		query += " AND flat.neighborhoods_count >= $" + fmt.Sprint(filtersCount)
		params = append(params, filters["neighborhoods_count_from"])
		filtersCount++
	}

	if filters["neighborhoods_count_to"] != nil {
		query += " AND flat.neighborhoods_count <= $" + fmt.Sprint(filtersCount)
		params = append(params, filters["neighborhoods_count_to"])
		filtersCount++
	}

	if filters["neighborhood_age_from"] != nil {
		query += " AND flat.neighborhood_age_from >= $" + fmt.Sprint(filtersCount)
		params = append(params, filters["neighborhood_age_from"])
		filtersCount++
	}

	if filters["neighborhood_age_to"] != nil {
		query += " AND flat.neighborhood_age_to <= $" + fmt.Sprint(filtersCount)
		params = append(params, filters["neighborhood_age_to"])
		filtersCount++
	}

	if filters["sex"] != nil {
		query += " AND flat.sex = $" + fmt.Sprint(filtersCount)
		params = append(params, filters["sex"])
		filtersCount++
	}

	if filters["created_by_id"] != nil {
		query += " AND flat.created_by_id = $" + fmt.Sprint(filtersCount)
		params = append(params, filters["created_by_id"])
		filtersCount++
	}

	params = append(params, offset, limit)
	query += fmt.Sprintf(` ORDER BY flat.up_in_search DESC, flat.created_at DESC OFFSET $%d LIMIT $%d;`, filtersCount, filtersCount+1)
	rows, err := flatRepo.Pool.Query(ctx, query, params...)
	if err != nil {
		return nil, customerror.NewError("flatRepo.GetFlats", flatRepo.Host+":"+flatRepo.Port, err.Error())
	}
	for rows.Next() {
		var flat flat.Flat
		var user user.User
		err := rows.Scan(
			&flat.Id,
			&flat.Name,
			&flat.About,
			&flat.PriceFrom,
			&flat.PriceTo,
			&flat.NeighborhoodsCount,
			&flat.NeighborhoodAgeFrom,
			&flat.NeighborhoodAgeTo,
			&flat.Sex,
			&flat.CreatedAt,
			&flat.CreatedById,
			&flat.UpInSearch,
			&user.UUID,
			&user.Firstname,
			&user.Lastname,
			&user.AvatarUrl,
		)
		if err != nil {
			return nil, customerror.NewError("flatRepo.GetFlats", flatRepo.Host+":"+flatRepo.Port, err.Error())
		}
		flat.CreatedByUser = user
		flats = append(flats, flat)
	}
	return flats, nil
}

func (flatRepo *FlatRepository) GetFlat(ctx context.Context, id int64) (*flat.Flat, error) {
	var flat flat.Flat
	query := `SELECT flat.id, flat.name, flat.about, flat.price_from, flat.price_to, flat.neighborhoods_count, 
	flat.neighborhood_age_from, flat.neighborhood_age_to, flat.sex,
	flat.created_at, flat.created_by_id, flat.up_in_search, users.id, users.firstname, users.lastname, users.avatar_url 
	FROM flat JOIN users ON flat.created_by_id = users.id WHERE flat.id = $1`
	row := flatRepo.Pool.QueryRow(ctx, query, id)
	err := row.Scan(
		&flat.Id,
		&flat.Name,
		&flat.About,
		&flat.PriceFrom,
		&flat.PriceTo,
		&flat.NeighborhoodsCount,
		&flat.NeighborhoodAgeFrom,
		&flat.NeighborhoodAgeTo,
		&flat.Sex,
		&flat.CreatedAt,
		&flat.CreatedById,
		&flat.UpInSearch,
		&flat.CreatedByUser.UUID,
		&flat.CreatedByUser.Firstname,
		&flat.CreatedByUser.Lastname,
		&flat.CreatedByUser.AvatarUrl,
	)
	if err == pgx.ErrNoRows {
		return nil, pgx.ErrNoRows
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, customerror.NewError("flatRepo.GetFlat", flatRepo.Host+":"+flatRepo.Port, err.Error())
	}
	return &flat, nil
}

func (flatRepo *FlatRepository) InsertFlat(ctx context.Context, flat *flat.Flat) (int64, error) {
	query := `INSERT INTO flat (name, about, price_from, price_to, neighborhoods_count, neighborhood_age_from, neighborhood_age_to, sex, created_by_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`
	var id int64
	err := flatRepo.Pool.QueryRow(ctx, query, flat.Name, flat.About, flat.PriceFrom, flat.PriceTo, flat.NeighborhoodsCount, flat.NeighborhoodAgeFrom, flat.NeighborhoodAgeTo, flat.Sex, flat.CreatedById).Scan(&id)
	if err != nil {
		return 0, customerror.NewError("flatRepo.InsertFlat", flatRepo.Host+":"+flatRepo.Port, err.Error())
	}
	return id, nil
}

func (flatRepo *FlatRepository) UpdateFlat(ctx context.Context, flat *flat.Flat, user *user.User) error {
	query := `UPDATE flat SET name = $1, about = $2, price_from = $3, price_to = $4, neighborhoods_count = $5, neighborhood_age_from = $6, neighborhood_age_to = $7, sex = $8 WHERE id = $9`
	whereArgs := []any{flat.Name, flat.About, flat.PriceFrom, flat.PriceTo, flat.NeighborhoodsCount, flat.NeighborhoodAgeFrom, flat.NeighborhoodAgeTo, flat.Sex, flat.Id}
	if !user.IsSuperUser {
		query += ` AND created_by_id = $10`
		whereArgs = append(whereArgs, user.UUID)
	}
	command, err := flatRepo.Pool.Exec(ctx, query, whereArgs...)
	if err != nil {
		return customerror.NewError("flatRepo.UpdateFlat", flatRepo.Host+":"+flatRepo.Port, err.Error())
	}
	if command.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (flatRepo *FlatRepository) DeleteFlat(ctx context.Context, id int64, user *user.User) error {
	args := []any{id}
	query := `DELETE FROM flat WHERE id = $1`
	if !user.IsSuperUser {
		query += ` AND created_by_id = $2`
		args = append(args, user.UUID)
	}
	command, err := flatRepo.Pool.Exec(ctx, query, args...)
	if err != nil {
		return customerror.NewError("flatRepo.DeleteFlat", flatRepo.Host+":"+flatRepo.Port, err.Error())
	}
	if command.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (flatRepo *FlatRepository) GetFlatImages(ctx context.Context, flatId int64) ([]flat.FlatImage, error) {
	query := `SELECT id, flat_id, url, filename FROM flat_image WHERE flat_id = $1`
	rows, err := flatRepo.Pool.Query(ctx, query, flatId)
	if err != nil {
		return nil, customerror.NewError("flatRepo.GetFlatImages", flatRepo.Host+":"+flatRepo.Port, err.Error())
	}
	var flatImages []flat.FlatImage
	for rows.Next() {
		var flatImage flat.FlatImage
		err := rows.Scan(&flatImage.Id, &flatImage.FlatId, &flatImage.Url, &flatImage.Filename)
		if err != nil {
			return nil, customerror.NewError("flatRepo.GetFlatImages", flatRepo.Host+":"+flatRepo.Port, err.Error())
		}
		flatImages = append(flatImages, flatImage)
	}
	return flatImages, nil
}
func (flatRepo *FlatRepository) InsertFlatImage(ctx context.Context, flatImage *flat.FlatImage) error {
	query := `INSERT INTO flat_image (flat_id, url, filename) VALUES ($1, $2, $3)`
	_, err := flatRepo.Pool.Exec(ctx, query, flatImage.FlatId, flatImage.Url, flatImage.Filename)
	if err != nil {
		return customerror.NewError("flatRepo.InsertFlatImage", flatRepo.Host+":"+flatRepo.Port, err.Error())
	}
	return nil
}
func (flatRepo *FlatRepository) DeleteFlatImage(ctx context.Context, flatImage *flat.FlatImage) error {
	query := `DELETE FROM flat_image WHERE id = $1`
	_, err := flatRepo.Pool.Exec(ctx, query, flatImage.Id)
	if err != nil {
		return customerror.NewError("flatRepo.DeleteFlatImage", flatRepo.Host+":"+flatRepo.Port, err.Error())
	}
	return nil
}
