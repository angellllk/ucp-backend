package main

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/session"
	"log"
	"os"
	"os/signal"
	config "sarp_backend/config"
	"sarp_backend/handler"
	"sarp_backend/repository"
	"sarp_backend/service"
	"syscall"
	"time"
)

func StartServer() {
	cfg, errRead := config.Read("./cfg.json")
	if errRead != nil {
		log.Fatalf("error reading cfg.json: %v", errRead)
	}

	logFileName := "log_" + time.Now().Format("2006-01-02_15-04-05") + ".log"
	loggerService, err := service.NewLoggerService(logFileName, cfg.Version)
	if err != nil {
		log.Fatalf("error creating logger: %v", err)
	}
	defer loggerService.Shutdown()

	ucpRepo, errRepo := repository.New(cfg.Dsn)
	if errRepo != nil {
		log.Fatalf("error creating repository: %v", errRepo)
		return
	}

	userService := service.NewUserService(ucpRepo)
	charService := service.NewCharacterService(ucpRepo)
	emailService := service.NewEmailService(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPFrom)
	authService := service.NewAuthService(session.New(session.Config{
		CookieSecure:   true,
		CookieHTTPOnly: true,
		CookieSameSite: "Strict",
	}))
	authMiddleware := service.NewMiddleware(authService)

	ucpHandler := handler.New(userService, charService, authService, loggerService, emailService)

	fiberConfig := fiber.Config{
		BodyLimit:               4 * 1024 * 10,
		Concurrency:             1024,
		ReadTimeout:             5 * time.Second,
		WriteTimeout:            5 * time.Second,
		ReadBufferSize:          4 * 1024 * 10,
		WriteBufferSize:         4 * 1024 * 10,
		Prefork:                 false,
		EnableTrustedProxyCheck: true,
		TrustedProxies:          []string{"127.0.0.1", "::1"},
	}
	app := fiber.New(fiberConfig)
	app.Use(logger.New(), compress.New())

	app.Use(cors.New(cors.Config{
		AllowMethods: "GET,POST",
		AllowHeaders: "Origin, Content-Type, Accept",
		AllowOrigins: "https://app.ro",
	}))

	app.Use(limiter.New(limiter.Config{
		Max:        500,
		Expiration: 1 * time.Hour,
		KeyGenerator: func(ctx *fiber.Ctx) string {
			realIP := ctx.Get("X-Real-IP")
			if realIP == "" {
				realIP = ctx.IP()
			}
			return realIP
		},
		LimitReached: func(ctx *fiber.Ctx) error {
			ip := ctx.Get("X-Real-IP")
			if ip == "" {
				ip = ctx.IP()
			}
			loggerService.Info(fmt.Sprintf("Rate limit reached for IP: %s", ip))
			return ctx.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":   true,
				"message": "You've reached the limit of HTTP requests. Try again later.",
			})
		},
	}))

	// Serve static files from the "build" directory
	app.Static("/", cfg.FEPath)

	app.Get("/join", func(ctx *fiber.Ctx) error {
		html := `
		<!DOCTYPE html>
		<html lang="en">
		<head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
		</head>
		<body>
			<h1>Joining</h1>
		</body>
		</html>
		`
		return ctx.Type("html").SendString(html)
	})

	SetupRoutes(app, authMiddleware, ucpHandler)

	// Route for 404
	app.Get("/*", func(c *fiber.Ctx) error {
		if _, err := os.Stat(cfg.FEPath + "/index.html"); err != nil {
			return c.Status(404).SendString("Not Found")
		}
		return c.SendFile(cfg.FEPath + "/index.html")
	})

	// Start the server
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	loggerService.Info(fmt.Sprintf("Starting server on %s\n", cfg.Port))
	go func() {
		if err = app.Listen(cfg.Port); err != nil {
			loggerService.Exception(fmt.Sprintf("error starting server: %v", err))
			os.Exit(1)
		}
	}()

	done := make(chan struct{})

	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				retentionPeriod := 7 * 24 * time.Hour
				if err = loggerService.ClearOldLogs(retentionPeriod); err != nil {
					loggerService.Exception(fmt.Sprintf("Error cleaning old logs: %v\n", err))
				}
			case <-done:
				loggerService.Info("Stopping log cleanup ticker.")
			}
		}
	}()

	<-stop

	loggerService.Info("Shutting down server...")
	if err = app.Shutdown(); err != nil {
		loggerService.Exception(fmt.Sprintf("error during shutdown: %v", err))
	}

	close(done)
	os.Exit(1)
}

func SetupRoutes(app *fiber.App, authMiddleware *service.Middleware, ucpHandler *handler.UserHandler) {
	api := app.Group("internal-ucp-api")

	v1 := api.Group("v1")

	v1.Use("/register", authMiddleware.EnsureLoggedOut)
	v1.Post("/register", ucpHandler.Register) // Apel direct la handler

	v1.Use("/confirm", authMiddleware.EnsureLoggedOut)
	v1.Get("/confirm", ucpHandler.Confirm)

	v1.Use("/reset-request", authMiddleware.EnsureLoggedOut)
	v1.Get("/reset-request", ucpHandler.ResetRequest)

	v1.Use("/confirm-reset", authMiddleware.EnsureLoggedOut)
	v1.Get("/confirm-reset", ucpHandler.ConfirmReset)

	v1.Use("/update-password", authMiddleware.EnsureLoggedOut)
	v1.Post("/update-password", ucpHandler.UpdatePassword)

	v1.Use("/login", authMiddleware.EnsureLoggedOut)
	v1.Post("/login", ucpHandler.Login)

	v1.Use("/logout", authMiddleware.EnsureAuthenticated)
	v1.Post("/logout", ucpHandler.Logout)

	v1.Use("/check-auth", authMiddleware.EnsureAuthenticated)
	v1.Get("/check-auth", ucpHandler.CheckAuth)

	v1.Use("/get-data", authMiddleware.EnsureAuthenticated)
	v1.Get("/get-data", ucpHandler.GetStats)

	v1.Use("/get-staff", authMiddleware.EnsureAuthenticated)
	v1.Get("/get-staff", ucpHandler.GetStaff)

	v1.Use("/server-stats", authMiddleware.EnsureAuthenticated)
	v1.Get("/server-stats", ucpHandler.ServerStats)

	v1.Use("/create-character", authMiddleware.EnsureAuthenticated)
	v1.Post("/create-character", ucpHandler.CreateCharacter)

	v1.Use("/restricted", func(ctx *fiber.Ctx) error {
		return authMiddleware.EnsurePrivilege(ctx)
	})

	v1.Get("/restricted/check", ucpHandler.CheckAdmin)
	v1.Get("/restricted/waiting-list", ucpHandler.WaitingList)
	v1.Post("/restricted/accept-character", ucpHandler.AcceptCharacter)
	v1.Post("/restricted/reject-character", ucpHandler.RejectCharacter)
	v1.Post("/restricted/fetch-character", ucpHandler.FetchCharacter)
	v1.Get("/restricted/ban-list", ucpHandler.BanList)
	v1.Post("/restricted/ban", ucpHandler.Ban)
	v1.Post("/restricted/unban", ucpHandler.Unban)
	v1.Post("/restricted/ajail", ucpHandler.Ajail)
	v1.Post("/restricted/logs", ucpHandler.Logs)
}
