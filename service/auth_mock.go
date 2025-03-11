package service

import (
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/mock"
)

type MockAuthService struct {
	mock.Mock
}

func (a *MockAuthService) CheckSession(ctx *fiber.Ctx) (string, bool, bool, error) {
	args := a.Called(ctx)
	return args.String(0), args.Bool(1), args.Bool(2), args.Error(3)
}

func (a *MockAuthService) SaveSession(ctx *fiber.Ctx, name string, isTester bool, isAdmin bool) error {
	args := a.Called(ctx, name, isTester, isAdmin)
	return args.Error(0)
}

func (a *MockAuthService) DestroySession(ctx *fiber.Ctx) error {
	args := a.Called(ctx)
	return args.Error(0)
}

func (a *MockAuthService) Authenticate(ctx *fiber.Ctx) error {
	return nil
}
