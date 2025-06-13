package flat

import (
	"mymate/pkg/user"
	"time"

	"github.com/google/uuid"
)

type Flat struct {
	Id                  int64     `json:"id"`
	Name                string    `json:"name"`
	About               string    `json:"about"`
	PriceFrom           uint64    `json:"price_from"`
	PriceTo             uint64    `json:"price_to"`
	NeighborhoodsCount  uint32    `json:"neighborhoods_count"`
	NeighborhoodAgeFrom uint32    `json:"neightborhood_age_from"`
	NeighborhoodAgeTo   uint32    `json:"neightborhood_age_to"`
	Sex                 string    `json:"sex"`
	CreatedAt           time.Time `json:"created_at"`
	CreatedById         uuid.UUID `json:"created_by_id"`
	CreatedByUser       user.User `json:"user"`
	UpInSearch          int       `json:"up_in_search"`
}

type FlatImage struct {
	Id       int64  `json:"id"`
	FlatId   int64  `json:"flat_id"`
	Url      string `json:"url"`
	Filename string `json:"filename"`
}
