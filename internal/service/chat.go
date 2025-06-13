package service

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"mymate/internal/repository"
	chatmessages "mymate/pkg/chat_messages"
	"mymate/pkg/customerror"
	"mymate/pkg/user"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type ChatServiceI interface {
	Connect(ctx *gin.Context, user *user.User, authErr error) error
	ServeWebSocket(connection *websocket.Conn)
	SendToUser(message *chatmessages.ChatMessage)
	GetChats(user *user.User) ([]repository.ChatWithUser, error)
	GetMessages(whatUser uuid.UUID, withUser uuid.UUID, fromMessage int64, offset int64, limit int64) ([]chatmessages.ChatMessage, error)
	KeepAlive()
}

type ChatService struct {
	Connections sync.Map
	ChatRepo    repository.ChatRepositoryI
	UserRepo    repository.UserRepositoryI
	Upgrader    websocket.Upgrader
	Host        string
	Port        string
}

func NewChatService(chatRepo repository.ChatRepositoryI, userRepo repository.UserRepositoryI, host string, port string) ChatServiceI {
	return &ChatService{
		Connections: sync.Map{},
		ChatRepo:    chatRepo,
		UserRepo:    userRepo,
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		Host: host,
		Port: port,
	}
}

func (s *ChatService) Connect(ctx *gin.Context, user *user.User, authErr error) error {
	connection, err := s.Upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		return customerror.NewError("chatService.Connect", s.Host+":"+s.Port, err.Error())
	}
	if authErr == jwt.ErrTokenExpired {
		connection.WriteJSON(gin.H{
			"status": http.StatusUnauthorized,
			"body":   gin.H{},
			"error":  "token expired",
		})
	}
	if authErr != nil {
		connection.Close()
		return customerror.NewError("chatService.Connect", s.Host+":"+s.Port, authErr.Error())
	}
	s.Connections.Store(connection, user)
	go s.ServeWebSocket(connection)
	return nil
}

type WebsocketMessage struct {
	Receiver string `json:"receiver_id"`
	Message  string `json:"message"`
}

func (s *ChatService) ServeWebSocket(connection *websocket.Conn) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("recovered from panic in ServeWebSocket: %v", r)
		}
		connection.Close()
		s.Connections.Delete(connection)
	}()
	for {
		var message WebsocketMessage
		err := connection.ReadJSON(&message)
		if err != nil {
			fmt.Println(err)
			connection.Close()
			s.Connections.Delete(connection)
			return
		}
		receiverUUID, err := uuid.Parse(message.Receiver)
		if err != nil {
			fmt.Println(err)
			connection.Close()
			s.Connections.Delete(connection)
			return
		}
		senderInteface, ok := s.Connections.Load(connection)
		if !ok {
			fmt.Println("sender not found")
			connection.Close()
			s.Connections.Delete(connection)
			return
		}
		sender := senderInteface.(*user.User)
		messageId, err := s.ChatRepo.AddMessage(context.Background(), sender.UUID, receiverUUID, message.Message)
		if err != nil {
			fmt.Println(err)
			connection.Close()
			s.Connections.Delete(connection)
			return
		}
		fmt.Println(&chatmessages.ChatMessage{
			Id:         messageId,
			SenderId:   sender.UUID,
			ReceiverId: receiverUUID,
			Message:    message.Message,
			CreatedAt: sql.NullTime{
				Time:  time.Now(),
				Valid: true,
			},
		})
		s.SendToUser(&chatmessages.ChatMessage{
			Id:         messageId,
			SenderId:   sender.UUID,
			ReceiverId: receiverUUID,
			Message:    message.Message,
			CreatedAt: sql.NullTime{
				Time:  time.Now(),
				Valid: true,
			},
		})
	}
}
func (s *ChatService) SendToUser(message *chatmessages.ChatMessage) {
	s.Connections.Range(func(key, value any) bool {
		connection := key.(*websocket.Conn)
		valueUser := value.(*user.User)
		if valueUser.UUID != message.ReceiverId {
			return false
		}
		err := connection.WriteJSON(message)
		if err != nil {
			connection.Close()
			s.Connections.Delete(connection)
			return false
		}
		return true
	})
}

func (s *ChatService) KeepAlive() {
	var deadCandidates sync.Map
	for {
		deadCandidates.Range(func(key, value any) bool {
			if _, ok := s.Connections.Load(key); ok {
				deadCandidates.Delete(key)
				return false
			}
			retries := value.(int)
			if retries > 10 {
				key := key.(*websocket.Conn)
				key.Close()
				s.Connections.Delete(key)
				deadCandidates.Delete(key)
				return false
			}
			deadCandidates.Store(key, retries+1)
			return true
		})
		s.Connections.Range(func(key, value any) bool {
			connection := key.(*websocket.Conn)
			err := connection.WriteMessage(websocket.PingMessage, nil)
			if err != nil {
				if _, ok := deadCandidates.Load(key); !ok {
					deadCandidates.Store(key, 1)
				}
				return false
			}
			if _, ok := deadCandidates.Load(key); ok {
				deadCandidates.Delete(key)
			}
			return true
		})
		time.Sleep(10 * time.Second)
	}
}

func (s *ChatService) GetChats(user *user.User) ([]repository.ChatWithUser, error) {
	chats, err := s.ChatRepo.GetChats(context.Background(), user)
	if err != nil {
		err := err.(customerror.CustomError)
		err.AppendModule("ChatService.GetChats")
		return nil, err
	}
	return chats, nil
}

func (s *ChatService) GetMessages(whatUser uuid.UUID, withUser uuid.UUID, fromMessage int64, offset int64, limit int64) ([]chatmessages.ChatMessage, error) {
	ctx, close := context.WithTimeout(context.Background(), time.Minute)
	defer close()
	messages, err := s.ChatRepo.GetMessages(ctx, whatUser, withUser, fromMessage, offset, limit)
	if err != nil {
		err := err.(customerror.CustomError)
		err.AppendModule("ChatService.GetMessages")
		return nil, err
	}
	return messages, nil
}
