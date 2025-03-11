package service

import (
	"github.com/gofiber/fiber/v2"
	"sarp_backend/model"
)

type UserServiceInterface interface {
	Create(data *model.RegisterAPI) error
	ActivateAccount(email string) error
	CheckActivation(name string) (bool, error)
	CheckForBan(name string) (bool, error)
	Verify(data *model.LoginAPI) error
	UpdatePassword(email string, password string) error
	Fetch(name string, email string) (bool, error)
	FetchMail(name string) (string, error)
	IsTester(name string) (bool, error)
	IsAdmin(name string) (bool, error)
	GetStats(name string) (*model.GetStatsAPI, error)
	GetStaff() ([]model.GetStaffAPI, error)
	GetServerStats() (*model.ServerStatsAPI, error)
	FetchCharacter(name string) (*model.CharacterAPI, error)
	Ban(data *model.BanAPI) error
	BanList() ([]model.BanAPI, error)
	Unban(data *model.BanAPI) error
	Ajail(data *model.AjailAPI) error
	Logs(data *model.LogsAPI) ([]map[string]interface{}, error)
	DeleteExpired() error
}

type AuthServiceInterface interface {
	CheckSession(ctx *fiber.Ctx) (string, bool, bool, error)
	SaveSession(ctx *fiber.Ctx, name string, isTester bool, isAdmin bool) error
	DestroySession(ctx *fiber.Ctx) error
	Authenticate(ctx *fiber.Ctx) error
}

type CharacterServiceInterface interface {
	Create(data *model.CharacterDataAPI) error
	FetchWaiting() ([]model.CharacterDataAPI, error)
	AcceptCharacter(data model.CharacterAPI) error
	DeclineCharacter(data model.RejectCharacterAPI) error
}

type LoggerInterface interface {
	Info(msg string)
	Warning(msg string)
	Exception(msg string)
	Debug(msg string)
	Shutdown()
}

type EmailInterface interface {
	SendEmail(to, subject, body string) error
}
