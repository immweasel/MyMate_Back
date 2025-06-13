package chatmessages

import (
	"database/sql"

	"github.com/google/uuid"
)

type ChatMessage struct {
	Id         int64        `json:"id"`
	CreatedAt  sql.NullTime `json:"created_at"`
	Message    string       `json:"message"`
	SenderId   uuid.UUID    `json:"sender_id"`
	ReceiverId uuid.UUID    `json:"receiver_id"`
}
