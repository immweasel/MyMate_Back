package security

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
)

func HashPassword(password string, salt string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(salt+password)))
}

func GenerateOTP() string {
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}

func GenerateHash() string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%d", rand.Intn(1000000)))))
}
