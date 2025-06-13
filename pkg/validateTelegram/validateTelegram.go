package validatetelegram

import (
	"encoding/json"
	"net/url"
	"time"

	"github.com/heyqbnk/twa-init-data-golang"
)

type TelegramUser struct {
	ID           int64  `json:"id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Username     string `json:"username"`
	LanguageCode string `json:"language_code"`
}

func ValidateTelegramData(initData string, botToken string) (*TelegramUser, bool) {
	params, _ := url.ParseQuery(initData)
	expIn := 24 * time.Hour
	isValid, err := twa.Validate(initData, botToken, expIn)
	if err != nil {
		return nil, false
	}
	if !isValid {
		return nil, false
	}
	userData := params.Get("user")

	if userData == "" {
		return nil, false
	}
	var user TelegramUser
	if err := json.Unmarshal([]byte(userData), &user); err != nil {
		return nil, false
	}
	return &user, true
}
