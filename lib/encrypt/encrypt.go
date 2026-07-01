package encrypt

import (
	"strings"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
)

func GenerateFromPassword(password *string) error {
	encryptedPassword, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	*password = string(encryptedPassword)
	return err
}

func CompareHashAndPassword(hashedPassword, password *string) error {
	return bcrypt.CompareHashAndPassword([]byte(*hashedPassword), []byte(*password))
}

func NewWithClaims(claims jwt.Claims) (ss string, err error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, err = token.SignedString([]byte("erp-cbqa-global"))
	return
}

func Parse(auth string) (tokenRaw string, claims jwt.MapClaims, err error) {
	ss := strings.ReplaceAll(auth, "Bearer ", "")
	token, err := jwt.Parse(ss, func(token *jwt.Token) (interface{}, error) {
		return []byte("erp-cbqa-global"), nil
	})
	if err != nil {
		return
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return token.Raw, claims, nil
	}
	return
}
