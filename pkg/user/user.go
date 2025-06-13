package user

import (
	"database/sql"

	"github.com/google/uuid"
)

type User struct {
	UUID               uuid.UUID    `json:"uuid"`
	Email              string       `json:"email"`
	IsActive           bool         `json:"is_active"`
	OTP                string       `json:"-"`
	OTPCreatedAt       sql.NullTime `json:"-"`
	OTPAttempts        int32        `json:"otp_attempts"`
	ResetHash          string       `json:"-"`
	ResetHashCreatedAt sql.NullTime `json:"-"`
	ResetHashAttempts  int32        `json:"reset_hash_attempts"`
	TelegramId         int64        `json:"telegram_id"`
	Firstname          string       `json:"firstname"`
	Lastname           string       `json:"lastname"`
	AvatarUrl          string       `json:"avatar_url"`
	AvatarFileName     string       `json:"avatar_file_name"`
	PasswordHash       string       `json:"-"`
	Birthdate          sql.NullTime `json:"birthdate"`
	Status             string       `json:"status"`
	EducationPlace     string       `json:"education_place"`
	EducationLevel     string       `json:"education_level"`
	About              string       `json:"about"`
	JWTVersion         uint         `json:"jwt_version"`
	IsSuperUser        bool         `json:"is_superuser"`
	Amount             uint64       `json:"amount"`
}
