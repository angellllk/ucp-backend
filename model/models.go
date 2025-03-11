package model

import (
	"errors"
	"net/mail"
	"regexp"
	"time"
)

type BaseResponse struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
}

type RegisterAPI struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r *RegisterAPI) Validate() error {
	if r.Username == "" || r.Email == "" || r.Password == "" {
		return errors.New("one or more fields are empty")
	}

	validUsername := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	if !validUsername.MatchString(r.Username) {
		return errors.New("username can only contain letters and digits")
	}

	if _, err := mail.ParseAddress(r.Email); err != nil {
		return errors.New("invalid email address")
	}

	if len(r.Password) < 8 {
		return errors.New("password length must be greater than 8")
	}

	if !containsLetter(r.Password) {
		return errors.New("password must contain at least one letter")
	}

	if !containsDigit(r.Password) {
		return errors.New("password must contain at least one digit")
	}

	if !containsSpecialChar(r.Password) {
		return errors.New("password must contain at least one special character")
	}

	return nil
}

type LoginAPI struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (l *LoginAPI) Validate() error {
	if l.Username == "" || l.Password == "" {
		return errors.New("one or more fields are empty")
	}

	validUsername := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	if !validUsername.MatchString(l.Username) {
		return errors.New("username can only contain letters and digits")
	}

	return nil
}

type UpdatePassword struct {
	Email       string `json:"email"`
	Token       string `json:"token"`
	Timestamp   int64  `json:"timestamp"`
	NewPassword string `json:"new_password"`
}

func (r *UpdatePassword) Validate() error {
	if _, err := mail.ParseAddress(r.Email); err != nil {
		return errors.New("invalid email address")
	}

	if len(r.NewPassword) < 8 {
		return errors.New("password length must be greater than 8")
	}

	if !containsLetter(r.NewPassword) {
		return errors.New("password must contain at least one letter")
	}

	if !containsDigit(r.NewPassword) {
		return errors.New("password must contain at least one digit")
	}

	if !containsSpecialChar(r.NewPassword) {
		return errors.New("password must contain at least one special character")
	}

	return nil
}

type CharacterStatsAPI struct {
	Name         string `json:"character_name"`
	Created      int    `json:"character_status"`
	Level        int    `json:"character_level"`
	PlayingHours int    `json:"playing_hours"`
}

type GetStatsAPI struct {
	Username      string              `json:"username"`
	Admin         int                 `json:"admin"`
	Tester        int                 `json:"tester"`
	DonateRank    int                 `json:"donate_rank"`
	Characters    int                 `json:"characters"`
	LastLogin     int64               `json:"last_login"`
	CharacterList []CharacterStatsAPI `json:"character_list"`
}

type GetStaffAPI struct {
	Username string `json:"username"`
	Role     string `json:"role"`
}

type CharacterDataAPI struct {
	Username        string `json:"username"`
	CharacterName   string `json:"character_name"`
	CharacterAge    int    `json:"character_age"`
	CharacterGender int    `json:"character_gender"`
	CharacterOrigin string `json:"character_origin"`
}

func (c *CharacterDataAPI) Validate() error {
	if c.CharacterName == "" {
		return errors.New("name cannot be empty")
	}

	if !checkCharacterName(c.CharacterName) {
		return errors.New("invalid character name")
	}

	if c.CharacterOrigin == "" {
		return errors.New("origin cannot be empty")
	}

	if !(c.CharacterAge > 12 && c.CharacterAge < 80) {
		return errors.New("age must be between 12 and 80 years old")
	}

	if containsDigit(c.CharacterOrigin) || containsSpecialChar(c.CharacterOrigin) {
		return errors.New("origin contains wrong characters")
	}

	if len(c.CharacterOrigin) < 4 {
		return errors.New("invalid length for character origin")
	}

	return nil
}

type CharacterAPI struct {
	Username      string `json:"username"`
	CharacterName string `json:"character_name"`
	AcceptedBy    string
}

type RejectCharacterAPI struct {
	Username      string `json:"username"`
	CharacterName string `json:"character_name"`
	Reason        string `json:"reason"`
}

type BanAPI struct {
	Username   string         `json:"username"`
	Expire     uint           `json:"expire"`
	Reason     string         `json:"reason"`
	AdminName  string         `json:"admin"`
	Characters []CharacterAPI `json:"characters"`
}

type AjailAPI struct {
	Character string `json:"character"`
	AdminName string `json:"admin"`
	Time      int    `json:"time"`
	Reason    string `json:"reason"`
}

type ServerStatsAPI struct {
	Online     int `json:"players_online"`
	Bans       int `json:"total_bans"`
	Houses     int `json:"total_houses"`
	Staff      int `json:"total_staff"`
	Accounts   int `json:"total_accounts"`
	Characters int `json:"total_characters"`
}

type LogsAPI struct {
	Type string `json:"type"`
}

type LogEntry struct {
	ID        int         `json:"id"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}
