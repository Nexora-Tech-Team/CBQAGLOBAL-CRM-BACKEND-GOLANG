package form

import (
	"unicode"
)

func ValidatePasswordCMS(s string) bool {
	var hasNumber, hasUpperCase, hasLowercase, hasSpecial bool
	for _, c := range s {
		switch {
		case unicode.IsNumber(c):
			hasNumber = true
		case unicode.IsUpper(c):
			hasUpperCase = true
		case unicode.IsLower(c):
			hasLowercase = true
		case unicode.IsPunct(c) || unicode.IsSymbol(c):
			hasSpecial = true
		}
	}
	return hasNumber && hasUpperCase && hasLowercase && hasSpecial
}
