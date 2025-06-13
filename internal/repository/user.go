package repository

import (
	"context"
	"errors"
	"fmt"
	"mymate/pkg/config"
	"mymate/pkg/customerror"
	"mymate/pkg/user"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepositoryI interface {
	CreateTables(ctx context.Context) error
	GetUser(ctx context.Context, id uuid.UUID) (*user.User, error)
	GetUserByCredentials(ctx context.Context, field string, value any) (*user.User, error)
	UpdateUser(ctx context.Context, user *user.User) error
	UpdateUserSensetive(ctx context.Context, user *user.User) error
	InsertUser(ctx context.Context, user *user.User) error
}

type UserRepository struct {
	Pool *pgxpool.Pool
	Host string
	Port string
}

func NewUserRepository(pool *pgxpool.Pool, appConfig *config.Config) UserRepositoryI {

	return &UserRepository{
		Pool: pool,
		Host: appConfig.WebHost,
		Port: appConfig.WebPort,
	}
}

func (userRepo *UserRepository) CreateTables(ctx context.Context) error {
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS users (
		id              UUID PRIMARY KEY,
		email           TEXT DEFAULT '',
		telegram_id     BIGINT DEFAULT 0,
		firstname       TEXT DEFAULT '',
		lastname        TEXT DEFAULT '',
		avatar_url      TEXT DEFAULT '',
		password_hash   TEXT DEFAULT '',
		avatar_file_name TEXT DEFAULT '',
		birthdate       DATE,
		status          TEXT DEFAULT '',
		education_place TEXT DEFAULT '',
		education_level TEXT DEFAULT '',
		about           TEXT DEFAULT '',
		jwt_version 	INTEGER DEFAULT 0,
		is_superuser    BOOLEAN DEFAULT FALSE,
		amount          BIGINT DEFAULT 0,
		otp             TEXT DEFAULT '',
		otp_created_at  TIMESTAMP,
		otp_attempts    INTEGER DEFAULT 0,
		reset_hash      TEXT DEFAULT '',
		reset_hash_created_at TIMESTAMP,
		reset_hash_attempts INTEGER DEFAULT 0,
		is_active       BOOLEAN DEFAULT FALSE
	);`
	_, err := userRepo.Pool.Exec(ctx, createTableQuery)
	if err != nil {
		return customerror.NewError("userRepo.CreateTables", userRepo.Host+":"+userRepo.Port, err.Error())
	}
	createIndexQuery := `CREATE INDEX IF NOT EXISTS user_id_idx ON users(id);`
	_, err = userRepo.Pool.Exec(ctx, createIndexQuery)
	if err != nil {
		return customerror.NewError("userRepo.CreateTables", userRepo.Host+":"+userRepo.Port, err.Error())
	}
	return nil
}
func (userRepo *UserRepository) GetUser(ctx context.Context, id uuid.UUID) (*user.User, error) {
	var user user.User
	query := `SELECT id, email, telegram_id, firstname, lastname, avatar_url, birthdate, status, education_place, education_level, about,jwt_version, avatar_file_name, is_superuser, amount, otp, otp_created_at, reset_hash, reset_hash_created_at, is_active, reset_hash_attempts, otp_attempts FROM users WHERE id=$1`
	err := userRepo.Pool.QueryRow(ctx, query, id).Scan(
		&user.UUID,
		&user.Email,
		&user.TelegramId,
		&user.Firstname,
		&user.Lastname,
		&user.AvatarUrl,
		&user.Birthdate,
		&user.Status,
		&user.EducationPlace,
		&user.EducationLevel,
		&user.About,
		&user.JWTVersion,
		&user.AvatarFileName,
		&user.IsSuperUser,
		&user.Amount,
		&user.OTP,
		&user.OTPCreatedAt,
		&user.ResetHash,
		&user.ResetHashCreatedAt,
		&user.IsActive,
		&user.ResetHashAttempts,
		&user.OTPAttempts,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, customerror.NewError("userRepo.GetUser", userRepo.Host+":"+userRepo.Port, err.Error())
	}
	return &user, nil
}

/*
Функция потенциально уязвима через field.
field подставлять только вручную, не давать пользователю его задавать!
*/
func (userRepo *UserRepository) GetUserByCredentials(ctx context.Context, field string, value any) (*user.User, error) {
	var user user.User
	query := fmt.Sprintf(`SELECT id, email, telegram_id, firstname, lastname, avatar_url, birthdate, status, education_place, education_level, about, jwt_version, avatar_file_name, is_superuser, amount, otp, otp_created_at, reset_hash, reset_hash_created_at, is_active, reset_hash_attempts, otp_attempts, password_hash FROM users WHERE %s`, field) + `=$1`
	err := userRepo.Pool.QueryRow(ctx, query, value).Scan(
		&user.UUID,
		&user.Email,
		&user.TelegramId,
		&user.Firstname,
		&user.Lastname,
		&user.AvatarUrl,
		&user.Birthdate,
		&user.Status,
		&user.EducationPlace,
		&user.EducationLevel,
		&user.About,
		&user.JWTVersion,
		&user.AvatarFileName,
		&user.IsSuperUser,
		&user.Amount,
		&user.OTP,
		&user.OTPCreatedAt,
		&user.ResetHash,
		&user.ResetHashCreatedAt,
		&user.IsActive,
		&user.ResetHashAttempts,
		&user.OTPAttempts,
		&user.PasswordHash,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, customerror.NewError("userRepo.GetUser", userRepo.Host+":"+userRepo.Port, err.Error())
	}
	return &user, nil
}

func (userRepo *UserRepository) UpdateUserSensetive(ctx context.Context, user *user.User) error {
	query := `UPDATE users SET email=$1, password_hash=$2, jwt_version=$3 WHERE id=$4`
	_, err := userRepo.Pool.Exec(ctx, query,
		user.Email,
		user.PasswordHash,
		user.JWTVersion,
		user.UUID,
	)
	if err != nil {
		return customerror.NewError("userRepo.UpdateUserSensetive", userRepo.Host+":"+userRepo.Port, err.Error())
	}
	return nil
}

func (userRepo *UserRepository) UpdateUser(ctx context.Context, user *user.User) error {

	query := `UPDATE users SET  
		firstname=$1, 
		lastname=$2, 
		avatar_url=$3, 
		birthdate=$4, 
		status=$5, 
		education_place=$6, 
		education_level=$7, 
		about=$8,
		jwt_version=$9,
		avatar_file_name=$10,
		otp=$11,
		otp_created_at=$12,
		reset_hash=$13,
		reset_hash_created_at=$14,
		is_active=$15,
		otp_attempts=$16,
		reset_hash_attempts=$17
		WHERE id=$18`
	command, err := userRepo.Pool.Exec(ctx, query,
		user.Firstname,
		user.Lastname,
		user.AvatarUrl,
		user.Birthdate,
		user.Status,
		user.EducationPlace,
		user.EducationLevel,
		user.About,
		user.JWTVersion,
		user.AvatarFileName,
		user.OTP,
		user.OTPCreatedAt,
		user.ResetHash,
		user.ResetHashCreatedAt,
		user.IsActive,
		user.OTPAttempts,
		user.ResetHashAttempts,
		user.UUID,
	)
	fmt.Print(err)
	if command.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	if err != nil {
		return customerror.NewError("userRepo.UpdateUser", userRepo.Host+":"+userRepo.Port, err.Error())
	}
	return nil
}

func (userRepo *UserRepository) InsertUser(ctx context.Context, user *user.User) error {
	query := `INSERT INTO users (id, email, telegram_id, firstname, lastname, avatar_url, birthdate, status, education_place, education_level, about, avatar_file_name, is_active, otp, otp_created_at, otp_attempts, password_hash) 
	VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,$12,$13, $14, $15, $16, $17)`
	command, err := userRepo.Pool.Exec(ctx, query,
		user.UUID,
		user.Email,
		user.TelegramId,
		user.Firstname,
		user.Lastname,
		user.AvatarUrl,
		user.Birthdate,
		user.Status,
		user.EducationPlace,
		user.EducationLevel,
		user.About,
		user.AvatarFileName,
		user.IsActive,
		user.OTP,
		user.OTPCreatedAt,
		user.OTPAttempts,
		user.PasswordHash,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				return customerror.ErrUUIDAlreadyExists
			}
		}
		return customerror.NewError("userRepo.InsertUser", userRepo.Host+":"+userRepo.Port, err.Error())
	}
	if command.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}
