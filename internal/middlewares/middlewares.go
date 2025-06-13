package middlewares

import (
	"context"
	"errors"
	"log"
	"mymate/internal/repository"
	"mymate/internal/service"
	"mymate/pkg/customerror"
	"net/http"
	"strconv"
	"time"

	"mymate/pkg/user"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
)

type MiddlewaresI interface {
	ValidUser() gin.HandlerFunc
	ThisUserOrAdmin() gin.HandlerFunc
	MyFlat() gin.HandlerFunc
}

type Middlewares struct {
	jwtService service.JWTServiceI
	userRepo   repository.UserRepositoryI
	flatRepo   repository.FlatRepositoryI
	host       string
	port       string
}

func NewMiddlewares(jwtService service.JWTServiceI, userRepo repository.UserRepositoryI, host, port string, flatRepo repository.FlatRepositoryI) MiddlewaresI {
	return &Middlewares{
		jwtService: jwtService,
		userRepo:   userRepo,
		host:       host,
		port:       port,
		flatRepo:   flatRepo,
	}
}
func (middlewares *Middlewares) ValidUser() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")
		user, err := middlewares.jwtService.ValidateToken(authHeader)
		if errors.Is(err, jwt.ErrTokenExpired) {
			ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
				"status": http.StatusUnauthorized,
				"body":   gin.H{},
				"error":  "token expired",
			})
			return
		}
		if err == customerror.ErrJwtInvalid || err == customerror.ErrJwtVersionIncorrect || err == pgx.ErrNoRows {
			ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
				"status": http.StatusUnauthorized,
				"body":   gin.H{},
				"error":  "token invalid",
			})
			return
		}
		if err != nil {
			customErr := err.(customerror.CustomError)
			customErr.AppendModule("Middlewares")
			log.Print(customErr.Error())
			ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
				"status": http.StatusInternalServerError,
				"body":   gin.H{},
				"error":  "Internal Server Error",
			})
			return
		}
		ctx.Set("user", user)
		ctx.Next()
	}
}
func (middlewares *Middlewares) ThisUserOrAdmin() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reqId := ctx.Param("id")
		authUser, exists := ctx.Get("user")
		if !exists {
			ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
				"status": http.StatusInternalServerError,
				"body":   gin.H{},
				"error":  "Internal Server Error",
			})
			return
		}
		user := authUser.(*user.User)
		if !user.IsSuperUser && reqId != user.UUID.String() {
			ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
				"status": http.StatusForbidden,
				"body":   gin.H{},
				"error":  "Forbidden",
			})
			return
		}
		ctx.Next()
	}
}

func (middlewares *Middlewares) MyFlat() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authUser, exists := ctx.Get("user")
		if !exists {
			ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
				"status": http.StatusInternalServerError,
				"body":   gin.H{},
				"error":  "Internal Server Error",
			})
			return
		}
		user := authUser.(*user.User)

		flatIdStr := ctx.Param("id")
		flatId, err := strconv.ParseInt(flatIdStr, 10, 64)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
				"status": http.StatusBadRequest,
				"body":   gin.H{},
				"error":  "invalid id",
			})
			return
		}
		c, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		flat, err := middlewares.flatRepo.GetFlat(c, flatId)
		if err == pgx.ErrNoRows {
			ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
				"status": http.StatusNotFound,
				"body":   gin.H{},
				"error":  "flat not found",
			})
			return
		}
		if err != nil {
			customErr := err.(customerror.CustomError)
			customErr.AppendModule("Middlewares")
			log.Print(customErr.Error())
			ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
				"status": http.StatusInternalServerError,
				"body":   gin.H{},
				"error":  "Internal Server Error",
			})
			return
		}
		if flat.CreatedById != user.UUID && !user.IsSuperUser {
			ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
				"status": http.StatusForbidden,
				"body":   gin.H{},
				"error":  "Forbidden",
			})
			return
		}
		ctx.Set("flat", flat)
		ctx.Next()
	}
}
