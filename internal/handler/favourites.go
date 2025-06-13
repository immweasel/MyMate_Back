package handler

import (
	"log"
	"mymate/internal/middlewares"
	"mymate/internal/service"
	"mymate/pkg/customerror"
	"mymate/pkg/user"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type FavouritesHandlerI interface {
	RegisterRoutes(group *gin.RouterGroup)
	GetFavourites(c *gin.Context)
	AddToFavourites(c *gin.Context)
	RemoveFromFavourites(c *gin.Context)
}

type FavouritesHandler struct {
	favouriteService service.FavouritesServiceI
	middlewares      middlewares.MiddlewaresI
	flatService      service.FlatServiceI
}

func NewFavouritesHandler(favouriteService service.FavouritesServiceI, middlewares middlewares.MiddlewaresI, flatService service.FlatServiceI) FavouritesHandlerI {
	return &FavouritesHandler{
		favouriteService: favouriteService,
		middlewares:      middlewares,
		flatService:      flatService,
	}
}

func (h *FavouritesHandler) RegisterRoutes(group *gin.RouterGroup) {
	favouriteGroup := group.Group("/favourites")
	favouriteGroup.Use(h.middlewares.ValidUser())
	favouriteGroup.GET("/", h.GetFavourites)
	favouriteGroup.POST("/", h.AddToFavourites)
	favouriteGroup.DELETE("/:flat_id", h.RemoveFromFavourites)
}

func (h *FavouritesHandler) GetFavourites(c *gin.Context) {
	userInterface, exists := c.Get("user")
	if !exists {
		c.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		return
	}
	user := userInterface.(*user.User)
	limitStr := c.Query("limit")
	offsetStr := c.Query("offset")
	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil {
		limit = 20
	}
	offset, err := strconv.ParseInt(offsetStr, 10, 64)
	if err != nil {
		offset = 0
	}

	favourites, err := h.favouriteService.GetFavourites(offset, limit, user.UUID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		log.Println(err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": http.StatusOK,
		"body": gin.H{
			"favourites": favourites,
		},
		"error": "",
	})
}

type AddToFavouritesRequest struct {
	FlatID int64 `json:"flat_id" binding:"required"`
}

func (h *FavouritesHandler) AddToFavourites(c *gin.Context) {
	userInterface, exists := c.Get("user")
	if !exists {
		c.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		return
	}
	user := userInterface.(*user.User)
	var request AddToFavouritesRequest
	if err := c.ShouldBindBodyWithJSON(&request); err != nil {
		c.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "invalid data",
		})
		return
	}
	flat, err := h.flatService.GetFlat(request.FlatID)

	if err == pgx.ErrNoRows {
		c.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusNotFound,
			"body":   gin.H{},
			"error":  "flat not found",
		})
		return
	}
	if err != nil {
		c.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		err := err.(customerror.CustomError)
		err.AppendModule("AddToFavourites")
		log.Println(err.Error())
		return
	}

	id, err := h.favouriteService.InsertFavourite(flat, user)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		err := err.(customerror.CustomError)
		err.AppendModule("AddToFavourites")
		log.Println(err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": http.StatusOK,
		"body": gin.H{
			"id": id,
		},
		"error": "",
	})
}

func (h *FavouritesHandler) RemoveFromFavourites(c *gin.Context) {
	flatIdStr := c.Param("flat_id")
	flatId, err := strconv.ParseInt(flatIdStr, 10, 64)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "invalid id",
		})
		return
	}
	userInterface, exists := c.Get("user")
	if !exists {
		c.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		return
	}
	user := userInterface.(*user.User)
	err = h.favouriteService.DeleteFavourite(flatId, user)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		err := err.(customerror.CustomError)
		err.AppendModule("RemoveFromFavourites")
		log.Println(err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": http.StatusOK,
		"body":   gin.H{},
		"error":  "",
	})
}
