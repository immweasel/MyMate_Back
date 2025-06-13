package handler

import (
	"fmt"
	"log"
	"mymate/internal/middlewares"
	"mymate/internal/service"
	"mymate/pkg/user"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ChatHandlerI interface {
	RegisterRoutes(group *gin.RouterGroup)
	Connect(ctx *gin.Context)
	GetChats(ctx *gin.Context)
	GetMessages(ctx *gin.Context)
}

type ChatHandler struct {
	chatService service.ChatServiceI
	jwtService  service.JWTServiceI
	host        string
	port        string
	middlewares middlewares.MiddlewaresI
}

func NewChatHandler(chatService service.ChatServiceI, host string, port string, middlewares middlewares.MiddlewaresI, jwtService service.JWTServiceI) ChatHandlerI {
	return &ChatHandler{
		chatService: chatService,
		host:        host,
		port:        port,
		middlewares: middlewares,
		jwtService:  jwtService,
	}
}

func (h *ChatHandler) RegisterRoutes(group *gin.RouterGroup) {
	chats := group.Group("/chats")
	chats.GET("/", h.middlewares.ValidUser(), h.GetChats)
	chats.GET("/:user_id", h.middlewares.ValidUser(), h.GetMessages)
	chats.GET("/websocket", h.Connect)
}

func (h *ChatHandler) Connect(ctx *gin.Context) {
	token := ctx.Query("token")
	user, authErr := h.jwtService.ValidateToken(token)
	_ = h.chatService.Connect(ctx, user, authErr)

}

func (h *ChatHandler) GetChats(ctx *gin.Context) {
	userInterface, exists := ctx.Get("user")
	if !exists {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		return
	}
	user := userInterface.(*user.User)
	chats, err := h.chatService.GetChats(user)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		log.Println(err.Error())
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"status": http.StatusOK,
		"body": gin.H{
			"chats": chats,
		},
		"error": nil,
	})
}

func (h *ChatHandler) GetMessages(ctx *gin.Context) {
	userInterface, exists := ctx.Get("user")
	if !exists {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		return
	}
	user := userInterface.(*user.User)
	limitStr := ctx.Query("limit")
	offsetStr := ctx.Query("offset")
	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil {
		limit = 20
	}
	offset, err := strconv.ParseInt(offsetStr, 10, 64)
	if err != nil {
		offset = 0
	}
	from := ctx.Query("from")
	fromInt, err := strconv.ParseInt(from, 10, 64)
	if err != nil {
		fromInt = -1
	}
	userIDStr := ctx.Param("user_id")
	userId, err := uuid.Parse(userIDStr)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "invalid data",
		})
		return
	}
	fmt.Println(user.UUID, userId, fromInt, offset, limit)
	messages, err := h.chatService.GetMessages(user.UUID, userId, fromInt, offset, limit)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		log.Println(err.Error())
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"status": http.StatusOK,
		"body": gin.H{
			"messages": messages,
		},
		"error": nil,
	})
}
