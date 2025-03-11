package model

import (
	"strings"
	"unicode"
)

func containsLetter(s string) bool {
	for _, c := range s {
		if unicode.IsLetter(c) {
			return true
		}
	}
	return false
}

func containsDigit(s string) bool {
	for _, c := range s {
		if unicode.IsDigit(c) {
			return true
		}
	}
	return false
}

func containsSpecialChar(s string) bool {
	specialChars := "!@#$%^&*()-_=+[]{}|;:'\",.<>?/`~"
	for _, c := range s {
		for _, sc := range specialChars {
			if c == sc {
				return true
			}
		}
	}
	return false
}

func checkCharacterName(s string) bool {
	sep := strings.Split(s, "_")
	if len(sep) != 2 {
		return false
	}

	ok := true
	for _, str := range sep {
		firstChar := rune(str[0])
		if containsDigit(str) || containsSpecialChar(str) || unicode.IsLower(firstChar) {
			ok = false
		}
	}
	return ok
}
