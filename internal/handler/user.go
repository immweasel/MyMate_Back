package handler

import (
	"database/sql"
	"log"
	"mymate/internal/middlewares"
	"mymate/internal/service"
	"mymate/pkg/customerror"
	userModel "mymate/pkg/user"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type UserHandlerI interface {
	RegisterRoutes(group *gin.RouterGroup)
	GetUser(ctx *gin.Context)
	UpdateUser(ctx *gin.Context)
	UpdateAvatar(ctx *gin.Context)
	DeleteAvatar(ctx *gin.Context)
}

type UserHandler struct {
	userService service.UserServiceI
	host        string
	port        string
	middlewares middlewares.MiddlewaresI
}

func NewUserHandler(userService service.UserServiceI, host, port string, middlewares middlewares.MiddlewaresI) UserHandlerI {
	return &UserHandler{
		userService: userService,
		host:        host,
		port:        port,
		middlewares: middlewares,
	}
}

func (userHandler *UserHandler) RegisterRoutes(group *gin.RouterGroup) {
	users := group.Group("/users", userHandler.middlewares.ValidUser())
	users.GET("/:id", userHandler.GetUser)
	users.PATCH("/:id", userHandler.middlewares.ThisUserOrAdmin(), userHandler.UpdateUser)
	users.PATCH("/:id/avatar", userHandler.middlewares.ThisUserOrAdmin(), userHandler.UpdateAvatar)
	users.DELETE("/:id/avatar", userHandler.middlewares.ThisUserOrAdmin(), userHandler.DeleteAvatar)
}
func (userHandler *UserHandler) GetUser(ctx *gin.Context) {
	idStr := ctx.Param("id")
	if idStr == "me" {
		idStr = ctx.MustGet("user").(*userModel.User).UUID.String()
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "invalid id",
		})
	}
	user, err := userHandler.userService.GetUser(id)
	if err == pgx.ErrNoRows {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusNotFound,
			"body":   gin.H{},
			"error":  "user not found",
		})
		return
	}
	if err != nil {
		customErr := err.(customerror.CustomError)
		customErr.AppendModule("UserHandler.GetUser")
		log.Print(customErr.Error())
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		return
	}
	if user.AvatarUrl == "" {
		user.AvatarUrl = "/media/user_placeholder.png"
	}
	ctx.JSON(http.StatusOK, gin.H{
		"status": http.StatusOK,
		"body": gin.H{
			"user": user,
		},
		"error": "",
	})
}

type UserUpdateRequest struct {
	Firstname      string    `json:"firstname"`
	Lastname       string    `json:"lastname"`
	Birthdate      time.Time `json:"birthdate"`
	Status         string    `json:"status"`
	EducationPlace string    `json:"education_place"`
	EducationLevel string    `json:"education_level"`
	About          string    `json:"about"`
	Amount         uint64    `json:"amount"`
}

func (userHandler *UserHandler) UpdateUser(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "invalid id",
		})
	}
	user, err := userHandler.userService.GetUser(id)
	if err == pgx.ErrNoRows {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusNotFound,
			"body":   gin.H{},
			"error":  "user not found",
		})
		return
	}
	var userFromRequest UserUpdateRequest
	if err := ctx.ShouldBindBodyWithJSON(&userFromRequest); err != nil {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "invalid data",
		})
		return
	}
	var userPatch userModel.User
	userPatch.Firstname = userFromRequest.Firstname
	userPatch.Lastname = userFromRequest.Lastname
	userPatch.Status = userFromRequest.Status
	userPatch.EducationPlace = userFromRequest.EducationPlace
	userPatch.EducationLevel = userFromRequest.EducationLevel
	userPatch.AvatarFileName = user.AvatarFileName
	userPatch.AvatarUrl = user.AvatarUrl
	userPatch.About = userFromRequest.About
	userPatch.Birthdate = sql.NullTime{Time: userFromRequest.Birthdate, Valid: true}
	if userFromRequest.Birthdate.IsZero() {
		userPatch.Birthdate.Valid = false
	}
	userPatch.UUID = id
	err = userHandler.userService.UpdateUser(&userPatch)
	if err == pgx.ErrNoRows {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusNotFound,
			"body":   gin.H{},
			"error":  "user not found",
		})
		return
	}
	if err != nil {
		customErr := err.(customerror.CustomError)
		customErr.AppendModule("UserHandler.UpdateUser")
		log.Print(customErr.Error())
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"status": http.StatusOK,
		"body": gin.H{
			"user": user,
		},
		"error": "",
	})
}
func (userHandler *UserHandler) UpdateAvatar(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "invalid id",
		})
	}
	user, err := userHandler.userService.GetUser(id)
	if err == pgx.ErrNoRows {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusNotFound,
			"body":   gin.H{},
			"error":  "user not found",
		})
		return
	}
	file, err := ctx.FormFile("file")
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "invalid file",
		})
		return
	}
	err = userHandler.userService.SaveUserAvatar(user, file)
	if err != nil {
		customErr := err.(customerror.CustomError)
		customErr.AppendModule("UserHandler.UpdateAvatar")
		log.Print(customErr.Error())
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"status": http.StatusOK,
		"body":   gin.H{},
		"error":  "",
	})
}

func (userHandler *UserHandler) DeleteAvatar(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "invalid id",
		})
	}
	user, err := userHandler.userService.GetUser(id)
	if err == pgx.ErrNoRows {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusNotFound,
			"body":   gin.H{},
			"error":  "user not found",
		})
		return
	}
	err = userHandler.userService.DeleteAvatar(user)
	if err != nil {
		customErr := err.(customerror.CustomError)
		customErr.AppendModule("UserHandler.DeleteAvatar")
		log.Print(customErr.Error())
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"status": http.StatusOK,
		"body":   gin.H{},
		"error":  "",
	})
}
