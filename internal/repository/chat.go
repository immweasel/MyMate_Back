package repository

import (
	"context"
	chatmessages "mymate/pkg/chat_messages"
	"mymate/pkg/customerror"
	"mymate/pkg/user"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ChatWithUser struct {
	Chat chatmessages.ChatMessage `json:"chat"`
	User *user.User               `json:"user"`
}

type ChatRepositoryI interface {
	CreateTables(ctx context.Context) error
	GetChats(ctx context.Context, user *user.User) ([]ChatWithUser, error)
	GetMessages(ctx context.Context, whatUser uuid.UUID, withUser uuid.UUID, fromMessage int64, offset int64, limit int64) ([]chatmessages.ChatMessage, error)
	AddMessage(ctx context.Context, senderId uuid.UUID, receiverId uuid.UUID, message string) (int64, error)
}

type ChatRepository struct {
	Host           string
	Port           string
	Pool           *pgxpool.Pool
	UserRepository UserRepositoryI
}

func NewChatReposiroty(host string, port string, pool *pgxpool.Pool, userRepo UserRepositoryI) ChatRepositoryI {
	return &ChatRepository{
		Host:           host,
		Port:           port,
		Pool:           pool,
		UserRepository: userRepo,
	}
}

func (r *ChatRepository) CreateTables(ctx context.Context) error {
	query := `CREATE TABLE IF NOT EXISTS chat_messages (
		id BIGSERIAL PRIMARY KEY,
		sender_id UUID NOT NULL REFERENCES users(id),
		receiver_id UUID NOT NULL REFERENCES users(id),
		message TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`
	_, err := r.Pool.Exec(ctx, query)
	if err != nil {
		return customerror.NewError("ChatRepository.CreateTables", r.Host+":"+r.Port, err.Error())
	}
	return nil
}

func (r *ChatRepository) GetChats(ctx context.Context, user *user.User) ([]ChatWithUser, error) {
	query := `
		SELECT DISTINCT ON (
    		LEAST(sender_id, receiver_id),
    		GREATEST(sender_id, receiver_id)
		)
    		id,
    		sender_id,
    		receiver_id,
    		message,
    		created_at
		FROM 
		    chat_messages
		WHERE 
		    $1 IN (sender_id, receiver_id)
		ORDER BY 
		    LEAST(sender_id, receiver_id),
		    GREATEST(sender_id, receiver_id),
		    id DESC;
	`
	rows, err := r.Pool.Query(ctx, query, user.UUID)
	if err != nil {
		return nil, customerror.NewError("ChatRepository.GetChats", r.Host+":"+r.Port, err.Error())
	}
	var chats []ChatWithUser
	for rows.Next() {
		var chat ChatWithUser
		err := rows.Scan(&chat.Chat.Id, &chat.Chat.SenderId, &chat.Chat.ReceiverId, &chat.Chat.Message, &chat.Chat.CreatedAt)
		if err != nil {
			continue
		}
		if user.UUID == chat.Chat.SenderId {
			chat.User, err = r.UserRepository.GetUser(ctx, chat.Chat.ReceiverId)
			if err != nil {
				continue
			}
		} else {
			chat.User, err = r.UserRepository.GetUser(ctx, chat.Chat.SenderId)
			if err != nil {
				continue
			}
		}
		chats = append(chats, chat)
	}
	return chats, nil
}
func (r *ChatRepository) GetMessages(ctx context.Context, whatUser uuid.UUID, withUser uuid.UUID, fromMessage int64, offset int64, limit int64) ([]chatmessages.ChatMessage, error) {
	query := `
		SELECT
			id,
    		sender_id,
    		receiver_id,
    		message,
    		created_at
		FROM
			chat_messages
		WHERE (sender_id = $1 OR receiver_id=$1) AND (sender_id = $2 OR receiver_id=$2) AND id <= $3
		ORDER BY id DESC
		OFFSET $4
		LIMIT $5;
	`
	if fromMessage == -1 {
		query = `
		SELECT
			id,
    		sender_id,
    		receiver_id,
    		message,
    		created_at
		FROM
			chat_messages
		WHERE (sender_id = $1 OR receiver_id=$1) AND (sender_id = $2 OR receiver_id=$2) AND id != $3
		ORDER BY id DESC
		OFFSET $4
		LIMIT $5;
	`
	}
	rows, err := r.Pool.Query(ctx, query, whatUser, withUser, fromMessage, offset, limit)
	if err != nil {
		return nil, customerror.NewError("ChatRepository.GetMessages", r.Host+":"+r.Port, err.Error())
	}
	var messages []chatmessages.ChatMessage
	for rows.Next() {
		var message chatmessages.ChatMessage
		err := rows.Scan(&message.Id, &message.SenderId, &message.ReceiverId, &message.Message, &message.CreatedAt)
		if err != nil {
			continue
		}
		messages = append(messages, message)
	}
	return messages, nil
}

func (r *ChatRepository) AddMessage(ctx context.Context, senderId uuid.UUID, receiverId uuid.UUID, message string) (int64, error) {
	query := `
		INSERT INTO chat_messages (sender_id, receiver_id, message) VALUES ($1,$2,$3);
	`
	_, err := r.Pool.Exec(ctx, query, senderId, receiverId, message)
	if err != nil {
		return 0, customerror.NewError("ChatRepository.AddMessage", r.Host+":"+r.Port, err.Error())
	}
	return 0, nil
}
