package handler

import (
	"database/sql"
	"log"
	"mymate/internal/middlewares"
	"mymate/internal/service"
	"mymate/pkg/config"
	"mymate/pkg/customerror"
	"mymate/pkg/user"
	validatetelegram "mymate/pkg/validateTelegram"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type AuthHandlerI interface {
	RegisterRoutes(group *gin.RouterGroup)
	SignIn(ctx *gin.Context)
	SignUp(ctx *gin.Context)
}

type TelegramAuthHandler struct {
	tgAuthService service.TelegramAuthenticationServiceI
	jwtService    service.JWTServiceI
	config        *config.Config
}

func NewTelegramAuthHandler(tgAuthService service.TelegramAuthenticationServiceI, jwtService service.JWTServiceI, config *config.Config) AuthHandlerI {
	return &TelegramAuthHandler{
		tgAuthService: tgAuthService,
		jwtService:    jwtService,
		config:        config,
	}
}

func (tgAuthHandler *TelegramAuthHandler) RegisterRoutes(group *gin.RouterGroup) {
	tgGroup := group.Group("/telegram")
	tgGroup.POST("/sign-in", tgAuthHandler.SignIn)
	tgGroup.POST("/sign-up", tgAuthHandler.SignUp)
}

type RequestBodyAuthorizeTelegram struct {
	InitData string `json:"initData"`
}

func (tgAuthHandler *TelegramAuthHandler) SignIn(ctx *gin.Context) {
	var request RequestBodyAuthorizeTelegram
	if err := ctx.ShouldBindBodyWithJSON(&request); err != nil {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "invalid data",
		})
		return
	}

	tgUser, valid := validatetelegram.ValidateTelegramData(request.InitData, tgAuthHandler.config.TelegramBotToken)
	if !valid {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "invalid initData",
		})
		return
	}

	appUser, err := tgAuthHandler.tgAuthService.SignIn(tgUser.ID)
	if err == pgx.ErrNoRows {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusUnauthorized,
			"body":   gin.H{},
			"error":  "user not exists",
		})
		return
	}
	if err != nil {
		customError := err.(customerror.CustomError)
		customError.AppendModule("TelegramAuthHandler.SignIn")
		log.Printf("%s", customError.Error())
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		return
	}

	accessToken, err := tgAuthHandler.jwtService.GenerateToken(appUser, true)
	if err != nil {
		customError := customerror.NewError("TelegramAuthHandler.SignIn", tgAuthHandler.config.WebHost+":"+tgAuthHandler.config.WebPort, err.Error())
		log.Printf("%s", customError.Error())
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		return
	}
	refreshToken, err := tgAuthHandler.jwtService.GenerateToken(appUser, false)
	if err != nil {
		customError := customerror.NewError("TelegramAuthHandler.SignIn", tgAuthHandler.config.WebHost+":"+tgAuthHandler.config.WebPort, err.Error())
		log.Printf("%s", customError.Error())
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
			"access_token":  accessToken,
			"refresh_token": refreshToken,
		},
		"error": nil,
	})
}
func (tgAuthHandler *TelegramAuthHandler) SignUp(ctx *gin.Context) {
	var request RequestBodyAuthorizeTelegram
	if err := ctx.ShouldBindBodyWithJSON(&request); err != nil {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "invalid data",
		})
		return
	}

	tgUser, valid := validatetelegram.ValidateTelegramData(request.InitData, tgAuthHandler.config.TelegramBotToken)
	if !valid {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "invalid initData",
		})
		return
	}
	firstname := tgUser.FirstName
	lastname := tgUser.LastName
	if firstname == "" && lastname == "" {
		firstname = tgUser.Username
	}
	if firstname == "" && lastname == "" {
		firstname = "Гость"
	}
	appUser, err := tgAuthHandler.tgAuthService.SignUp(tgUser.ID, firstname, lastname)
	if err == customerror.ErrUserAlreadyExists {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusForbidden,
			"body":   gin.H{},
			"error":  "User Already Exists",
		})
		return
	}
	if err != nil {
		customError := customerror.NewError("TelegramAuthHandler.SignUp", tgAuthHandler.config.WebHost+":"+tgAuthHandler.config.WebPort, err.Error())
		log.Printf("%s", customError.Error())
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		return
	}
	accessToken, err := tgAuthHandler.jwtService.GenerateToken(appUser, true)
	if err != nil {
		customError := customerror.NewError("TelegramAuthHandler.SignIn", tgAuthHandler.config.WebHost+":"+tgAuthHandler.config.WebPort, err.Error())
		log.Printf("%s", customError.Error())
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		return
	}
	refreshToken, err := tgAuthHandler.jwtService.GenerateToken(appUser, false)
	if err != nil {
		customError := customerror.NewError("TelegramAuthHandler.SignIn", tgAuthHandler.config.WebHost+":"+tgAuthHandler.config.WebPort, err.Error())
		log.Printf("%s", customError.Error())
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
			"access_token":  accessToken,
			"refresh_token": refreshToken,
		},
		"error": nil,
	})
}

type MailAuthHandlerI interface {
	RegisterRoutes(group *gin.RouterGroup)
	SignIn(ctx *gin.Context)
	SignUp(ctx *gin.Context)
	ResetPassword(ctx *gin.Context)
	ResetMail(ctx *gin.Context)
	GetOTP(ctx *gin.Context)
	Activate(ctx *gin.Context)
	GetResetHash(ctx *gin.Context)
}

type MailAuthHandler struct {
	mailService service.MailAuthServiceI
	jwtService  service.JWTServiceI
	config      *config.Config
	middlewares middlewares.MiddlewaresI
}

func NewMailAuthHandler(mailService service.MailAuthServiceI, jwtService service.JWTServiceI, config *config.Config, middlewares middlewares.MiddlewaresI) MailAuthHandlerI {
	return &MailAuthHandler{
		mailService: mailService,
		jwtService:  jwtService,
		config:      config,
		middlewares: middlewares,
	}
}

func (h *MailAuthHandler) RegisterRoutes(group *gin.RouterGroup) {
	mailGroup := group.Group("/mail")
	mailGroup.POST("/sign-in", h.SignIn)
	mailGroup.POST("/sign-up", h.SignUp)
	mailGroup.POST("/reset-mail", h.middlewares.ValidUser(), h.ResetMail)
	mailGroup.POST("/get-otp", h.GetOTP)
	mailGroup.POST("/get-reset-hash/:id", h.GetResetHash)
	mailGroup.POST("/activate/:id", h.Activate)
	mailGroup.POST("/reset-password/:id", h.ResetPassword)
}

type SignInRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *MailAuthHandler) SignIn(ctx *gin.Context) {
	var request SignInRequest
	if err := ctx.ShouldBindBodyWithJSON(&request); err != nil {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "invalid data",
		})
		return
	}
	if request.Email == "" || request.Password == "" {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "invalid data",
		})
		return
	}
	user, err := h.mailService.SignIn(request.Email, request.Password)
	if err == customerror.ErrWrongCredentials || err == pgx.ErrNoRows {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusUnauthorized,
			"body":   gin.H{},
			"error":  "invalid credentials",
		})
		return
	}
	if err != nil {
		customError := customerror.NewError("MailAuthHandler.SignIn", h.config.WebHost+":"+h.config.WebPort, err.Error())
		log.Printf("%s", customError.Error())
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		return
	}

	if !user.IsActive {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusUnauthorized,
			"body":   gin.H{},
			"error":  "user is not active",
		})
		return
	}

	accessToken, err := h.jwtService.GenerateToken(user, true)
	if err != nil {
		customError := customerror.NewError("MailAuthHandler.SignIn", h.config.WebHost+":"+h.config.WebPort, err.Error())
		log.Printf("%s", customError.Error())
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		return
	}
	refreshToken, err := h.jwtService.GenerateToken(user, false)
	if err != nil {
		customError := customerror.NewError("MailAuthHandler.SignIn", h.config.WebHost+":"+h.config.WebPort, err.Error())
		log.Printf("%s", customError.Error())
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
			"access_token":  accessToken,
			"refresh_token": refreshToken,
		},
		"error": nil,
	})
}

type SignUpRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
	Birthdate string `json:"birthdate"`
}

func (h *MailAuthHandler) SignUp(ctx *gin.Context) {
	var request SignUpRequest
	if err := ctx.ShouldBindBodyWithJSON(&request); err != nil {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "invalid data",
		})
		return
	}
	if request.Email == "" || request.Password == "" || (request.Firstname == "" && request.Lastname == "") {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "invalid data",
		})
		return
	}
	birthdateTime, err := time.Parse("2006-01-02", request.Birthdate)
	if err != nil || birthdateTime.IsZero() || birthdateTime.After(time.Now()) {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "invalid data",
		})
		return
	}
	birthdate := sql.NullTime{Time: birthdateTime, Valid: true}
	user, err := h.mailService.SignUp(request.Email, request.Password, request.Firstname, request.Lastname, birthdate)
	if err == customerror.ErrUserAlreadyExists {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusConflict,
			"body":   gin.H{},
			"error":  "user already exists",
		})
		return
	}
	if err != nil {
		customError := customerror.NewError("MailAuthHandler.SignUp", h.config.WebHost+":"+h.config.WebPort, err.Error())
		log.Printf("%s", customError.Error())
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		return
	}
	if user == nil {
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
			"user_id": user.UUID,
		},
		"error": nil,
	})
}

type ActivateRequest struct {
	OTP string `json:"otp"`
}

func (h *MailAuthHandler) Activate(ctx *gin.Context) {
	var activateRequest ActivateRequest
	if err := ctx.ShouldBindBodyWithJSON(&activateRequest); err != nil {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "invalid data",
		})
		return
	}
	idStr := ctx.Param("id")
	uuid, err := uuid.Parse(idStr)
	if activateRequest.OTP == "" || err != nil {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "invalid data",
		})
		return
	}
	err = h.mailService.ValidateOTP(uuid, activateRequest.OTP)
	if err == pgx.ErrNoRows || err == customerror.ErrWrongCredentials {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusForbidden,
			"body":   gin.H{},
			"error":  "invalid data",
		})
	}
	if err == customerror.ErrAttemptsEnded {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "attempts ended",
		})
		return
	}
	if err == customerror.ErrTimedOut {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "timed out",
		})
		return
	}
	if err != nil {
		customError := err.(customerror.CustomError)
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		customError.AppendModule("MailAuthHandler.Activate")
		log.Printf("%s", customError.Error())
		return
	}
	user, err := h.mailService.ActivateUser(uuid)
	if err == customerror.ErrUserAlreadyActivated {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "user already activated",
		})
		return
	}
	if err == pgx.ErrNoRows {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusNotFound,
			"body":   gin.H{},
			"error":  "user not found",
		})
		return
	}
	if err != nil {
		customError := err.(customerror.CustomError)
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		customError.AppendModule("MailAuthHandler.Activate")
		log.Print(customError.Error())
		return
	}
	accessToken, err := h.jwtService.GenerateToken(user, false)
	if err != nil {
		customError := err.(customerror.CustomError)
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		customError.AppendModule("MailAuthHandler.Activate")
		log.Print(customError.Error())
		return
	}
	refreshToken, err := h.jwtService.GenerateToken(user, true)
	if err != nil {
		customError := err.(customerror.CustomError)
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		customError.AppendModule("MailAuthHandler.Activate")
		log.Print(customError.Error())
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

type ResetPasswordRequest struct {
	ResetHash string `json:"reset_hash"`
	Password  string `json:"password"`
}

func (h *MailAuthHandler) ResetPassword(ctx *gin.Context) {
	var resetPasswordRequest ResetPasswordRequest
	if err := ctx.ShouldBindBodyWithJSON(&resetPasswordRequest); err != nil {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "invalid data",
		})
		return
	}
	if resetPasswordRequest.Password == "" || resetPasswordRequest.ResetHash == "" {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "invalid data",
		})
		return
	}
	idStr := ctx.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "invalid data",
		})
		return
	}
	err = h.mailService.ValidateResetHash(id, resetPasswordRequest.ResetHash)
	if err == pgx.ErrNoRows {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusNotFound,
			"body":   gin.H{},
			"error":  "user not found",
		})
		return
	}
	if err == customerror.ErrAttemptsEnded {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusUnauthorized,
			"body":   gin.H{},
			"error":  "attempts ended",
		})
		return
	}
	if err == customerror.ErrTimedOut {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "wait 5 minutes before trying again",
		})
		return
	}
	if err == customerror.ErrWrongCredentials {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusUnauthorized,
			"body":   gin.H{},
			"error":  "invalid credentials",
		})
		return
	}
	if err != nil {
		customError := err.(customerror.CustomError)
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		customError.AppendModule("MailAuthHandler.ResetPassword")
		log.Print(customError.Error())
		return
	}
	err = h.mailService.ResetPassword(id, resetPasswordRequest.Password)
	if err == pgx.ErrNoRows {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusNotFound,
			"body":   gin.H{},
			"error":  "user not found",
		})
		return
	}
	if err != nil {
		customError := err.(customerror.CustomError)
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		customError.AppendModule("MailAuthHandler.ResetPassword")
		log.Print(customError.Error())
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"status": http.StatusOK,
		"body":   gin.H{},
		"error":  nil,
	})
}

type ResetMailRequest struct {
	Email     string `json:"email"`
	ResetHash string `json:"reset_hash"`
}

func (h *MailAuthHandler) ResetMail(ctx *gin.Context) {
	user := ctx.MustGet("user").(user.User)
	var request ResetMailRequest
	if err := ctx.ShouldBindBodyWithJSON(&request); err != nil {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "invalid data",
		})
		return
	}
	if request.Email == "" || request.ResetHash == "" {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "invalid data",
		})
		return
	}
	err := h.mailService.ValidateResetHash(user.UUID, request.ResetHash)
	if err == pgx.ErrNoRows {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusNotFound,
			"body":   gin.H{},
			"error":  "user not found",
		})
		return
	}
	if err == customerror.ErrTimedOut {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "wait 5 minutes before trying again",
		})
		return
	}
	if err == customerror.ErrAttemptsEnded {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusUnauthorized,
			"body":   gin.H{},
			"error":  "attempts ended",
		})
		return
	}
	if err != nil {
		customError := err.(customerror.CustomError)
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		customError.AppendModule("MailAuthHandler.ResetMail")
		log.Print(customError.Error())
		return
	}
	err = h.mailService.ResetEmail(user.UUID, request.Email)
	if err == pgx.ErrNoRows {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusNotFound,
			"body":   gin.H{},
			"error":  "user not found",
		})
		return
	}
	if err != nil {
		customError := err.(customerror.CustomError)
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		customError.AppendModule("MailAuthHandler.ResetMail")
		log.Print(customError.Error())
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"status": http.StatusOK,
		"body":   gin.H{},
		"error":  nil,
	})
}

type GetOTPRequest struct {
	Email string `json:"email"`
}

func (h *MailAuthHandler) GetOTP(ctx *gin.Context) {
	var getOTPRequest GetOTPRequest
	if err := ctx.ShouldBindBodyWithJSON(&getOTPRequest); err != nil {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "invalid data",
		})
		return
	}
	if getOTPRequest.Email == "" {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "invalid data",
		})
		return
	}
	user, err := h.mailService.SetNewOTPByEmail(getOTPRequest.Email)
	if err == pgx.ErrNoRows {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusNotFound,
			"body":   gin.H{},
			"error":  "user not found",
		})
		return
	}
	if err == customerror.ErrEmailNotSet {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "email not set",
		})
		return
	}
	if err == customerror.ErrTimedOut {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "wait 5 minutes before trying again",
		})
		return
	}
	if err != nil {
		customError := err.(customerror.CustomError)
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		customError.AppendModule("MailAuthHandler.GetOTP")
		log.Print(customError.Error())
		return
	}
	go h.mailService.SendOTP(user.Email, user.OTP)

	ctx.JSON(http.StatusOK, gin.H{
		"status": http.StatusOK,
		"body": gin.H{
			"user_id": user.UUID.String(),
		},
		"error": nil,
	})
}

type GetResetHashRequest struct {
	OTP string `json:"otp"`
}

func (h *MailAuthHandler) GetResetHash(ctx *gin.Context) {
	idStr := ctx.Param("id")
	uuid, err := uuid.Parse(idStr)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "invalid data",
		})
		return
	}
	var getResetHashRequest GetResetHashRequest
	if err := ctx.ShouldBindBodyWithJSON(&getResetHashRequest); err != nil {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "invalid data",
		})
		return
	}
	if getResetHashRequest.OTP == "" {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"body":   gin.H{},
			"error":  "invalid data",
		})
		return
	}
	err = h.mailService.ValidateOTP(uuid, getResetHashRequest.OTP)
	if err == pgx.ErrNoRows || err == customerror.ErrWrongCredentials {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusForbidden,
			"body":   gin.H{},
			"error":  "invalid data",
		})
		return
	}
	if err == customerror.ErrAttemptsEnded {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusForbidden,
			"body":   gin.H{},
			"error":  "attempts ended",
		})
		return
	}
	if err == customerror.ErrTimedOut {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusForbidden,
			"body":   gin.H{},
			"error":  "timed out",
		})
		return
	}
	if err != nil {
		customError := err.(customerror.CustomError)
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		customError.AppendModule("MailAuthHandler.GetResetHash")
		log.Print(customError.Error())
		return
	}
	user, err := h.mailService.SetNewResetHash(uuid)
	if err == pgx.ErrNoRows {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusNotFound,
			"body":   gin.H{},
			"error":  "user not found",
		})
		return
	}
	if err != nil {
		customError := err.(customerror.CustomError)
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"status": http.StatusInternalServerError,
			"body":   gin.H{},
			"error":  "Internal Server Error",
		})
		customError.AppendModule("MailAuthHandler.GetResetHash")
		log.Print(customError.Error())
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"status": http.StatusOK,
		"body": gin.H{
			"reset_hash": user.ResetHash,
		},
		"error": nil,
	})
}
