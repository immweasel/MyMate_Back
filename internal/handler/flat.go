package handler

import (
	"fmt"
	"log"
	"mymate/internal/middlewares"
	"mymate/internal/service"
	modelsFlat "mymate/pkg/flat"
	modelsUser "mymate/pkg/user"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type FlatHandlerI interface {
	RegisterRoutes(group *gin.RouterGroup)
	GetFlats(ctx *gin.Context)
	GetFlat(ctx *gin.Context)
	InsertFlat(ctx *gin.Context)
	UpdateFlat(ctx *gin.Context)
	DeleteFlat(ctx *gin.Context)
	GetFlatImages(ctx *gin.Context)
	InsertFlatImage(ctx *gin.Context)
	DeleteFlatImage(ctx *gin.Context)
}

type FlatHandler struct {
	flatService service.FlatServiceI
	host        string
	port        string
	middlewares middlewares.MiddlewaresI
}

func NewFlatHandler(flatService service.FlatServiceI, host, port string, middlewares middlewares.MiddlewaresI) FlatHandlerI {
	return &FlatHandler{
		flatService: flatService,
		host:        host,
		port:        port,
		middlewares: middlewares,
	}
}

func (flatHandler *FlatHandler) RegisterRoutes(group *gin.RouterGroup) {
	flatGroup := group.Group("/flats")
	flatGroup.Use(flatHandler.middlewares.ValidUser())
	flatGroup.GET("/", flatHandler.GetFlats)
	flatGroup.GET("/:id", flatHandler.GetFlat)
	flatGroup.POST("/", flatHandler.InsertFlat)
	flatGroup.PATCH("/:id", flatHandler.middlewares.MyFlat(), flatHandler.UpdateFlat)
	flatGroup.DELETE("/:id", flatHandler.middlewares.MyFlat(), flatHandler.DeleteFlat)
	flatGroup.GET("/:id/images", flatHandler.GetFlatImages)
	flatGroup.POST("/:id/images", flatHandler.middlewares.MyFlat(), flatHandler.InsertFlatImage)
	flatGroup.DELETE("/:id/images/:image_id", flatHandler.middlewares.MyFlat(), flatHandler.DeleteFlatImage)
}

func (flatHandler *FlatHandler) GetFlats(ctx *gin.Context) {
	offset := ctx.DefaultQuery("offset", "0")
	limit := ctx.DefaultQuery("limit", "10")
	limitInt, err := strconv.ParseInt(limit, 10, 64)
	if err != nil {
		limitInt = 10
	}
	offsetInt, err := strconv.ParseInt(offset, 10, 64)
	if err != nil {
		offsetInt = 0
	}
	filters := map[string]any{}
	name := ctx.DefaultQuery("name", "")
	if name == "" {
		filters["name"] = nil
	} else {
		filters["name"] = name
	}
	about := ctx.DefaultQuery("about", "")
	if about == "" {
		filters["about"] = nil
	} else {
		filters["about"] = about
	}
	priceForm := ctx.DefaultQuery("price_from", "")
	if priceForm == "" {
		filters["price_from"] = nil
	} else {
		filters["price_from"] = priceForm
	}
	priceTo := ctx.DefaultQuery("price_to", "")
	if priceTo == "" {
		filters["price_to"] = nil
	} else {
		filters["price_to"] = priceTo
	}
	neighborhoods_count_from := ctx.DefaultQuery("neighborhoods_count_from", "")
	if neighborhoods_count_from == "" {
		filters["neighborhoods_count_from"] = nil
	} else {
		filters["neighborhoods_count_from"] = neighborhoods_count_from
	}
	neighborhoods_count_to := ctx.DefaultQuery("neighborhoods_count_to", "")
	if neighborhoods_count_to == "" {
		filters["neighborhoods_count_to"] = nil
	} else {
		filters["neighborhoods_count_to"] = neighborhoods_count_to
	}
	sex := ctx.DefaultQuery("sex", "")
	if sex == "" {
		filters["sex"] = nil
	} else {
		filters["sex"] = sex
	}
	createdById := ctx.DefaultQuery("created_by_id", "")
	if createdById == "" {
		filters["created_by_id"] = nil
	} else {
		filters["created_by_id"] = createdById
	}

	flats, err := flatHandler.flatService.GetFlats(offsetInt, limitInt, filters)
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
			"flats": flats,
		},
		"error": nil,
	})
}

func (flatHandler *FlatHandler) GetFlat(ctx *gin.Context) {
	id := ctx.Param("id")
	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "invalid id",
		})
		return
	}
	flat, err := flatHandler.flatService.GetFlat(idInt)
	if err == pgx.ErrNoRows {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusNotFound,
			"body":   gin.H{},
			"error":  "flat not found",
		})
		return
	}
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
			"flat": flat,
		},
		"error": nil,
	})
}

func (flatHandler *FlatHandler) InsertFlat(ctx *gin.Context) {
	userInt, exists := ctx.Get("user")
	if !exists {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		log.Print("user not found")
		return
	}
	user := userInt.(*modelsUser.User)

	var flatFromRequest modelsFlat.Flat
	if err := ctx.ShouldBindBodyWithJSON(&flatFromRequest); err != nil {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "invalid data",
		})
		return
	}
	flatFromRequest.CreatedByUser = *user
	flatFromRequest.CreatedById = user.UUID
	flatFromRequest.CreatedAt = time.Now()
	if flatFromRequest.Name == "" {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "name is required",
		})
		return
	}
	id, err := flatHandler.flatService.InsertFlat(&flatFromRequest)
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
			"id": id,
		},
		"error": nil,
	})
}
func (flatHandler *FlatHandler) UpdateFlat(ctx *gin.Context) {
	flatInt, exists := ctx.Get("flat")
	if !exists {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		log.Print("flat not found")
		return
	}
	flat := flatInt.(*modelsFlat.Flat)

	userInt, exists := ctx.Get("user")
	if !exists {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		log.Print("user not found")
		return
	}
	user := userInt.(*modelsUser.User)
	var flatFromRequest modelsFlat.Flat
	if err := ctx.ShouldBindBodyWithJSON(&flatFromRequest); err != nil {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "invalid data",
		})
		return
	}
	flatFromRequest.Id = flat.Id
	flatFromRequest.CreatedByUser = flat.CreatedByUser
	flatFromRequest.CreatedById = flat.CreatedById
	flatFromRequest.CreatedAt = flat.CreatedAt
	if flatFromRequest.Name == "" {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "name is required",
		})
		return
	}
	err := flatHandler.flatService.UpdateFlat(&flatFromRequest, user)
	if err == pgx.ErrNoRows {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusNotFound,
			"body":   gin.H{},
			"error":  "flat not found",
		})
		return
	}
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
		"body":   gin.H{},
		"error":  nil,
	})
}
func (flatHandler *FlatHandler) DeleteFlat(ctx *gin.Context) {
	flatInt, exists := ctx.Get("flat")
	fmt.Println("Hete3")
	if !exists {
		fmt.Println("Hete4")
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		log.Print("flat not found")
		return
	}
	fmt.Println("Hete5")
	flat := flatInt.(*modelsFlat.Flat)

	userInt, exists := ctx.Get("user")
	if !exists {
		fmt.Println("Hete6")
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		log.Print("user not found")
		return
	}
	user := userInt.(*modelsUser.User)
	err := flatHandler.flatService.DeleteFlat(flat.Id, user)
	if err == pgx.ErrNoRows {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusNotFound,
			"body":   gin.H{},
			"error":  "flat not found",
		})
		return
	}
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
		"body":   gin.H{},
		"error":  nil,
	})
}

func (fileHandler *FlatHandler) GetFlatImages(ctx *gin.Context) {
	flat := ctx.Param("id")

	flatNum, err := strconv.ParseInt(flat, 10, 64)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "invalid id",
		})
		return
	}

	images, err := fileHandler.flatService.GetFlatImages(flatNum)
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
			"images": images,
		},
		"error": nil,
	})
}
func (fileHandler *FlatHandler) InsertFlatImage(ctx *gin.Context) {
	flat, exists := ctx.Get("flat")
	if !exists {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		log.Print("flat not found")
		return
	}
	flatInt := flat.(*modelsFlat.Flat)
	file, err := ctx.FormFile("file")
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "invalid file",
		})
		return
	}
	err = fileHandler.flatService.InsertFlatImage(file, flatInt)
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
		"body":   gin.H{},
		"error":  nil,
	})
}
func (fileHandler *FlatHandler) DeleteFlatImage(ctx *gin.Context) {
	flat, exists := ctx.Get("flat")
	if !exists {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		return
	}
	flatInt := flat.(*modelsFlat.Flat)
	images, err := fileHandler.flatService.GetFlatImages(flatInt.Id)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		log.Print(err.Error())
		return
	}
	flatImageStr := ctx.Param("image_id")
	flatImageId, err := strconv.ParseInt(flatImageStr, 10, 64)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "invalid id",
		})
		return
	}
	for _, image := range images {
		if image.Id == flatImageId {
			err = fileHandler.flatService.DeleteFlatImage(&image)
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
				"body":   gin.H{},
				"error":  nil,
			})
			return
		}
	}
	ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
		"status": http.StatusNotFound,
		"body":   gin.H{},
		"error":  "image not found",
	})
}
