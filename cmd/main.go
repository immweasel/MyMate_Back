package main

import (
	"context"
	"fmt"
	"log"
	"mymate/internal/handler"
	"mymate/internal/middlewares"
	"mymate/internal/repository"
	"mymate/internal/service"
	"mymate/pkg/cleaner"
	"mymate/pkg/config"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robfig/cron/v3"
)

func initMonthlyCleaner(pool *pgxpool.Pool) {
	c := cron.New()

	// Запускать в 00:00 1-го числа каждого месяца
	_, err := c.AddFunc("0 0 1 * *", func() {
		cleaner.Clean(pool)
	})

	if err != nil {
		log.Fatalf("Failed to schedule cleanup job: %v", err)
	}

	go c.Start()

}

func main() {
	config, err := config.NewConfig(".env")
	if err != nil {
		log.Fatalf("%s", err.Error())
	}
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", config.DbUser, config.DbPassword, config.DbHost, config.DbPort, config.DbName)
	dbconfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		log.Fatalf("%s", err.Error())
	}
	dbconfig.MaxConns = 100
	dbconfig.MinConns = 10
	dbconfig.MaxConnLifetime = 1 * time.Hour
	dbconfig.MaxConnIdleTime = 15 * time.Minute
	pool, err := pgxpool.NewWithConfig(context.Background(), dbconfig)
	if err != nil {
		log.Fatalf("%s", err.Error())
	}
	defer pool.Close()
	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("%s", err.Error())
	}

	userRepository := repository.NewUserRepository(pool, config)
	flatRepository := repository.NewFlatRepository(pool, config.WebHost, config.WebPort, config.MainUrl)
	favouritesRepository := repository.NewFavouritesRepository(pool, config.WebHost, config.WebPort)
	chatRepository := repository.NewChatReposiroty(config.WebHost, config.WebPort, pool, userRepository)

	err = userRepository.CreateTables(context.Background())
	if err != nil {
		log.Fatal(err.Error())
	}
	err = flatRepository.CreateTables(context.Background())
	if err != nil {
		log.Fatal(err.Error())
	}
	err = favouritesRepository.CreateTables(context.Background())
	if err != nil {
		log.Fatal(err.Error())
	}
	err = chatRepository.CreateTables(context.Background())
	if err != nil {
		log.Fatal(err.Error())
	}
	initMonthlyCleaner(pool)

	tgAuthService := service.NewTelegramAuthService(userRepository, config.WebHost, config.WebPort)
	mailAuthService := service.NewMailAuthService(userRepository, config.WebHost, config.WebPort, config.MailToken, config.From, config.SecretKey)
	jwtService := service.NewJWTService(config, userRepository)
	middlewares := middlewares.NewMiddlewares(jwtService, userRepository, config.WebHost, config.WebPort, flatRepository)
	userService := service.NewUserService(userRepository, config.WebHost, config.WebPort, config.MainUrl)
	flatService := service.NewFlatService(flatRepository, config.WebHost, config.WebPort, config.MainUrl)
	favouritesService := service.NewFavouritesService(favouritesRepository, config.WebHost, config.WebPort)
	chatService := service.NewChatService(chatRepository, userRepository, config.WebHost, config.WebPort)
	go chatService.KeepAlive()
	tgAuthHandler := handler.NewTelegramAuthHandler(tgAuthService, jwtService, config)
	mailAuthHandler := handler.NewMailAuthHandler(mailAuthService, jwtService, config, middlewares)
	userHandler := handler.NewUserHandler(userService, config.WebHost, config.WebPort, middlewares)
	flatHandler := handler.NewFlatHandler(flatService, config.WebHost, config.WebPort, middlewares)
	favouritesHandler := handler.NewFavouritesHandler(favouritesService, middlewares, flatService)
	chatHandler := handler.NewChatHandler(chatService, config.WebHost, config.WebPort, middlewares, jwtService)

	initMonthlyCleaner(pool)

	router := gin.Default()
	api := router.Group("/api")
	v1 := api.Group("/v1")
	auth := v1.Group("/auth")
	auth.POST("/refresh-token", middlewares.ValidUser(), func(ctx *gin.Context) {
		handler.RefreshToken(ctx, jwtService)
	})

	tgAuthHandler.RegisterRoutes(auth)
	mailAuthHandler.RegisterRoutes(auth)
	userHandler.RegisterRoutes(v1)
	flatHandler.RegisterRoutes(v1)
	favouritesHandler.RegisterRoutes(v1)
	chatHandler.RegisterRoutes(v1)

	router.Run(config.WebHost + ":" + config.WebPort)
}
