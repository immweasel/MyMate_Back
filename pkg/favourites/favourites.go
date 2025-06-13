package favourites

import (
	"mymate/pkg/flat"

	"github.com/google/uuid"
)

type Favourites struct {
	Id     int64     `json:"id"`
	FlatId int64     `json:"flat_id"`
	UserId uuid.UUID `json:"user_id"`
	Flat   flat.Flat `json:"flat"`
}
