package handler

import (
	"log"
	"mymate/internal/service"
	modelsUser "mymate/pkg/user"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RefreshToken(ctx *gin.Context, jwtService service.JWTServiceI) {
	user, exists := ctx.Get("user")
	if !exists {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "invalid user",
		})
		return
	}
	accessToken, err := jwtService.GenerateToken(user.(*modelsUser.User), true)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		log.Print(err.Error())
		return
	}
	refreshToken, err := jwtService.GenerateToken(user.(*modelsUser.User), false)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		log.Print(err.Error())
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"status": http.StatusOK,
		"body": gin.H{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
		},
		"error": nil,
	})
}
