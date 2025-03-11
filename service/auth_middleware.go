package service

import (
	"github.com/gofiber/fiber/v2"
	"sarp_backend/model"
)

type Middleware struct {
	AuthService *AuthService
}

func NewMiddleware(authService *AuthService) *Middleware {
	return &Middleware{AuthService: authService}
}

func (m *Middleware) EnsureLoggedOut(ctx *fiber.Ctx) error {
	name, _, _, err := m.AuthService.CheckSession(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(model.BaseResponse{
			Error:   true,
			Message: "Error checking for session",
		})
	}

	if name != "" {
		return ctx.Status(fiber.StatusOK).JSON(model.BaseResponse{
			Error:   true,
			Message: "You are already logged in",
		})
	}
	return ctx.Next()
}

func (m *Middleware) EnsureAuthenticated(ctx *fiber.Ctx) error {
	name, _, _, err := m.AuthService.CheckSession(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(model.BaseResponse{
			Error:   true,
			Message: "Error checking for session",
		})
	}

	if name == "" {
		return ctx.Status(fiber.StatusOK).JSON(model.BaseResponse{
			Error:   true,
			Message: "You must be logged in to access this resource",
		})
	}
	return ctx.Next()
}

func (m *Middleware) EnsurePrivilege(ctx *fiber.Ctx) error {
	name, admin, tester, err := m.AuthService.CheckSession(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(model.BaseResponse{
			Error:   true,
			Message: "Error checking for session",
		})
	}

	if name == "" || (!admin && !tester) {
		return ctx.Status(fiber.StatusOK).JSON(model.BaseResponse{
			Error:   true,
			Message: "You must have correct privileges to access this resource",
		})
	}

	return ctx.Next()
}
