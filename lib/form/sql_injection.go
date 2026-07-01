package form

import (
	"fmt"
	"regexp"
	"strings"
)

func SQLInjector(input string) string {
	re := regexp.MustCompile(`['\"\n\r\t\;\$\^\*\\]|://`)
	input = re.ReplaceAllLiteralString(input, "")
	return input
}

func SQLInjectorNumber(input string) string {
	re := regexp.MustCompile(`[\Wa-zA-Z_]`)
	input = re.ReplaceAllLiteralString(input, "")
	return input
}

func SQLInjectorSingleNumber(input string) string {
	re := regexp.MustCompile(`[\Wa-zA-Z_]`)
	input = re.ReplaceAllLiteralString(input, "")
	if len(input) == 0 {
		return ""
	}
	return string(input[0])
}

func SQLInjectorNumberMultiple(input string) string {
	if input == "" {
		return ""
	} else {
		var result string
		re := regexp.MustCompile(`[\Wa-zA-Z_]`)
		tempInput := re.ReplaceAllLiteralString(input, "")
		splitInput := strings.Split(tempInput, "")
		result = "("
		for i, v := range splitInput {
			if i == 0 {
				result += fmt.Sprintf(`'%v'`, v)
			} else {
				result += fmt.Sprintf(`,'%v'`, v)
			}
		}
		result += ")"
		return result
	}
}
