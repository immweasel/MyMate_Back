package service

import (
	"context"
	"crypto/tls"
	"database/sql"
	"log"
	"mymate/internal/repository"
	"mymate/pkg/customerror"
	"mymate/pkg/security"
	"mymate/pkg/user"
	"net/smtp"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type MailAuthServiceI interface {
	SignIn(email string, password string) (*user.User, error)
	SignUp(email string, password string, firstname string, lastname string, birthdate sql.NullTime) (*user.User, error)
	ResetEmail(userId uuid.UUID, email string) error
	SendOTP(to string, otp string)
	ValidateOTP(userId uuid.UUID, otp string) error
	ValidateResetHash(userId uuid.UUID, resetHash string) error
	ResetPassword(userId uuid.UUID, password string) error
	ActivateUser(userId uuid.UUID) (*user.User, error)
	SetNewOTP(userId uuid.UUID) (*user.User, error)
	SetNewOTPByEmail(email string) (*user.User, error)
	SetNewResetHash(userId uuid.UUID) (*user.User, error)
}

type MailAuthService struct {
	userRepo  repository.UserRepositoryI
	host      string
	port      string
	mailToken string
	from      string
	salt      string
}

func NewMailAuthService(userRepo repository.UserRepositoryI, host, port string, mailToken, from string, salt string) MailAuthServiceI {
	return &MailAuthService{
		userRepo:  userRepo,
		host:      host,
		port:      port,
		mailToken: mailToken,
		from:      from,
		salt:      salt,
	}
}

func (mailService *MailAuthService) SetNewOTP(userId uuid.UUID) (*user.User, error) {
	ctx, close := context.WithTimeout(context.Background(), time.Minute)
	defer close()
	user, err := mailService.userRepo.GetUser(ctx, userId)
	if err == pgx.ErrNoRows {
		return nil, err
	}
	if err != nil {
		customError := err.(customerror.CustomError)
		customError.AppendModule("MailAuthenticationService.SetNewOTP")
		return nil, customError
	}
	if user.OTPCreatedAt.Time.Add(5 * time.Minute).After(time.Now()) {
		return nil, customerror.ErrTimedOut
	}
	user.OTP = security.GenerateOTP()
	user.OTPCreatedAt = sql.NullTime{
		Time:  time.Now(),
		Valid: true,
	}
	user.OTPAttempts = 5
	err = mailService.userRepo.UpdateUser(ctx, user)
	if err != nil {
		customError := err.(customerror.CustomError)
		customError.AppendModule("MailAuthenticationService.SetNewOTP")
		return nil, customError
	}
	return user, nil
}

func (mailService *MailAuthService) SetNewOTPByEmail(email string) (*user.User, error) {
	ctx, close := context.WithTimeout(context.Background(), time.Minute)
	defer close()
	user, err := mailService.userRepo.GetUserByCredentials(ctx, "email", email)
	if err == pgx.ErrNoRows {
		return nil, err
	}
	if err != nil {
		customError := err.(customerror.CustomError)
		customError.AppendModule("MailAuthenticationService.SetNewOTP")
		return nil, customError
	}
	if user.Email == "" {
		return nil, customerror.ErrEmailNotSet
	}
	if user.OTPCreatedAt.Time.Add(5 * time.Minute).After(time.Now()) {
		return nil, customerror.ErrTimedOut
	}
	user.OTP = security.GenerateOTP()
	user.OTPCreatedAt = sql.NullTime{
		Time:  time.Now(),
		Valid: true,
	}
	user.OTPAttempts = 5
	err = mailService.userRepo.UpdateUser(ctx, user)
	if err != nil {
		customError := err.(customerror.CustomError)
		customError.AppendModule("MailAuthenticationService.SetNewOTP")
		return nil, customError
	}
	return user, nil
}

func (mailService *MailAuthService) SetNewResetHash(userId uuid.UUID) (*user.User, error) {
	ctx, close := context.WithTimeout(context.Background(), time.Minute)
	defer close()
	user, err := mailService.userRepo.GetUser(ctx, userId)
	if err == pgx.ErrNoRows {
		return nil, err
	}
	if err != nil {
		customError := err.(customerror.CustomError)
		customError.AppendModule("MailAuthenticationService.SetNewResetHash")
		return nil, customError
	}
	user.ResetHash = security.GenerateHash()
	user.ResetHashCreatedAt = sql.NullTime{
		Time:  time.Now(),
		Valid: true,
	}
	user.ResetHashAttempts = 5
	err = mailService.userRepo.UpdateUser(ctx, user)
	if err != nil {
		customError := err.(customerror.CustomError)
		customError.AppendModule("MailAuthenticationService.SetNewResetHash")
		return nil, customError
	}
	return user, nil
}

func (mailService *MailAuthService) ActivateUser(userId uuid.UUID) (*user.User, error) {
	ctx, close := context.WithTimeout(context.Background(), time.Minute)
	defer close()
	user, err := mailService.userRepo.GetUser(ctx, userId)
	if err == pgx.ErrNoRows {
		return nil, err
	}
	if user.IsActive {
		return nil, customerror.ErrUserAlreadyActivated
	}
	if err != nil {
		customError := err.(customerror.CustomError)
		customError.AppendModule("MailAuthenticationService.ActivateUser")
		return nil, customError
	}
	user.IsActive = true
	err = mailService.userRepo.UpdateUser(ctx, user)
	if err != nil {
		customError := err.(customerror.CustomError)
		customError.AppendModule("MailAuthenticationService.ActivateUser")
		return nil, customError
	}
	return user, nil
}
func (mailService *MailAuthService) SignIn(email string, password string) (*user.User, error) {
	ctx, close := context.WithTimeout(context.Background(), time.Minute)
	defer close()
	user, err := mailService.userRepo.GetUserByCredentials(ctx, "email", email)
	if err == pgx.ErrNoRows {
		return nil, err
	}
	if err != nil {
		err := err.(customerror.CustomError)
		err.AppendModule("MailAuthenticationService.SignIn")
		return nil, err
	}
	if user.PasswordHash != security.HashPassword(password, mailService.salt) {
		log.Printf("User %s failed to sign in", user.Email)
		log.Printf("PasswordHash: %s", user.PasswordHash)
		log.Printf("Password: %s", security.HashPassword(password, mailService.salt))
		return nil, customerror.ErrWrongCredentials
	}
	return user, nil
}

func (mailService *MailAuthService) SignUp(email string, password string, firstname string, lastname string, birthdate sql.NullTime) (*user.User, error) {
	ctx, close := context.WithTimeout(context.Background(), time.Minute)
	defer close()
	_, err := mailService.userRepo.GetUserByCredentials(ctx, "email", email)
	if err == nil {
		return nil, customerror.ErrUserAlreadyExists
	}
	if err != pgx.ErrNoRows {
		customError := err.(customerror.CustomError)
		customError.AppendModule("MailAuthenticationService.SignUp")
		return nil, customError
	}
	retries := 0
	for retries < 10 {
		tempUUID, err := uuid.NewRandom()
		if err != nil {
			return nil, customerror.NewError("MailAuthenticationService.SignUp.GeneratingUUID", mailService.host+":"+mailService.port, err.Error())
		}
		var tempUser user.User = user.User{
			UUID:         tempUUID,
			Email:        email,
			PasswordHash: security.HashPassword(password, mailService.salt),
			Firstname:    firstname,
			Lastname:     lastname,
			Birthdate:    birthdate,
			IsActive:     false,
			OTP:          security.GenerateOTP(),
			OTPCreatedAt: sql.NullTime{
				Time:  time.Now(),
				Valid: true,
			},
			OTPAttempts: 5,
		}
		err = mailService.userRepo.InsertUser(ctx, &tempUser)
		if err == nil {
			go mailService.SendOTP(email, tempUser.OTP)
			return &tempUser, nil
		}
		retries++
	}
	return nil, customerror.ErrUserAlreadyExists
}
func (mailService *MailAuthService) ResetEmail(userId uuid.UUID, email string) error {
	ctx, close := context.WithTimeout(context.Background(), time.Minute)
	defer close()
	user, err := mailService.userRepo.GetUser(ctx, userId)
	if err == pgx.ErrNoRows {
		return err
	}
	if err != nil {
		customError := err.(customerror.CustomError)
		customError.AppendModule("MailAuthenticationService.ResetEmail")
		return customError
	}
	user.Email = email
	user.JWTVersion = user.JWTVersion + 1
	err = mailService.userRepo.UpdateUserSensetive(ctx, user)

	if err != nil {
		customError := err.(customerror.CustomError)
		customError.AppendModule("MailAuthenticationService.ResetEmail")
		return customError
	}
	return nil
}
func (mailService *MailAuthService) SendOTP(toMail string, otp string) {
	from := mailService.from
	password := mailService.mailToken

	to := []string{toMail}
	smtpHost := "smtp.mail.ru"
	smtpPort := "465"

	subject := "Subject: Благодарим за регистрацию на MyMate\r\n"
	body := "\nВаш пароль: " + otp + "\n"
	message := []byte(subject + "\r\n" + body)
	conn, err := tls.Dial("tcp", smtpHost+":"+smtpPort, &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         smtpHost,
	})
	if err != nil {
		log.Println(err)
		return
	}

	c, err := smtp.NewClient(conn, smtpHost)
	if err != nil {
		log.Println(err)
		return
	}

	auth := smtp.PlainAuth("", from, password, smtpHost)
	if err = c.Auth(auth); err != nil {
		log.Println(err)
		return
	}

	if err = c.Mail(from); err != nil {
		log.Println(err)
		return
	}

	for _, addr := range to {
		if err = c.Rcpt(addr); err != nil {
			log.Println(err)
			return
		}
	}

	w, err := c.Data()
	if err != nil {
		log.Println(err)
		return
	}

	_, err = w.Write(message)
	if err != nil {
		log.Println(err)
		return
	}

	err = w.Close()
	if err != nil {
		log.Println(err)
		return
	}
	c.Quit()
}
func (mailService *MailAuthService) ValidateOTP(userId uuid.UUID, otp string) error {
	ctx, close := context.WithTimeout(context.Background(), time.Minute)
	defer close()
	user, err := mailService.userRepo.GetUser(ctx, userId)
	if err == pgx.ErrNoRows {
		return err
	}
	if err != nil {
		customError := err.(customerror.CustomError)
		customError.AppendModule("MailAuthenticationService.ValidateOTP")
		return customError
	}
	if user.OTPAttempts == 0 {
		return customerror.ErrAttemptsEnded
	}
	if !user.OTPCreatedAt.Valid || user.OTP == "" || user.OTPCreatedAt.Time.Add(5*time.Minute).Before(time.Now()) {
		return customerror.ErrTimedOut
	}
	if user.OTP == otp {
		user.OTP = ""
		user.OTPCreatedAt.Valid = false
		user.OTPCreatedAt.Time = time.Time{}
		err = mailService.userRepo.UpdateUser(ctx, user)
		if err == pgx.ErrNoRows {
			return err
		}
		if err != nil {
			customError := err.(customerror.CustomError)
			customError.AppendModule("MailAuthenticationService.ValidateOTP")
			return customError
		}
		return nil
	}
	user.OTPAttempts = user.OTPAttempts - 1
	err = mailService.userRepo.UpdateUser(ctx, user)
	if err != nil {
		log.Println(err.Error())
	}
	return customerror.ErrWrongCredentials
}
func (mailService *MailAuthService) ResetPassword(userId uuid.UUID, password string) error {
	ctx, close := context.WithTimeout(context.Background(), time.Minute)
	defer close()
	user, err := mailService.userRepo.GetUser(ctx, userId)
	if err == pgx.ErrNoRows {
		return err
	}
	if err != nil {
		customError := err.(customerror.CustomError)
		customError.AppendModule("MailAuthenticationService.ResetPassword")
		return customError
	}
	user.PasswordHash = security.HashPassword(password, mailService.salt)
	user.JWTVersion = user.JWTVersion + 1
	err = mailService.userRepo.UpdateUserSensetive(ctx, user)
	if err == pgx.ErrNoRows {
		return err
	}
	if err != nil {
		customError := err.(customerror.CustomError)
		customError.AppendModule("MailAuthenticationService.ResetPassword")
		return customError
	}
	return nil
}

func (mailService *MailAuthService) ValidateResetHash(userId uuid.UUID, resetHash string) error {
	ctx, close := context.WithTimeout(context.Background(), time.Minute)
	defer close()
	user, err := mailService.userRepo.GetUser(ctx, userId)
	if err == pgx.ErrNoRows {
		return err
	}
	if err != nil {
		customError := err.(customerror.CustomError)
		customError.AppendModule("MailAuthenticationService.ResetPassword")
		return customError
	}
	if user.ResetHashAttempts == 0 {
		return customerror.ErrAttemptsEnded
	}
	if !user.ResetHashCreatedAt.Valid || user.ResetHash == "" || user.ResetHashCreatedAt.Time.Add(5*time.Minute).Before(time.Now()) {
		return customerror.ErrTimedOut
	}
	if user.ResetHash != resetHash {
		user.ResetHashAttempts = user.ResetHashAttempts - 1
		err = mailService.userRepo.UpdateUser(ctx, user)
		if err != nil {
			log.Println(err.Error())
		}
		return customerror.ErrWrongCredentials
	}
	user.ResetHash = ""
	user.ResetHashCreatedAt = sql.NullTime{
		Time:  time.Time{},
		Valid: false,
	}
	err = mailService.userRepo.UpdateUser(ctx, user)
	if err == pgx.ErrNoRows {
		return err
	}
	if err != nil {
		customError := err.(customerror.CustomError)
		customError.AppendModule("MailAuthenticationService.ResetPassword")
		return customError
	}
	return nil
}
