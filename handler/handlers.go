package handler

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"net/http"
	"net/mail"
	"sarp_backend/model"
	"sarp_backend/service"
	"strconv"
	"time"
)

type UserHandler struct {
	User        service.UserServiceInterface
	Char        service.CharacterServiceInterface
	Auth        service.AuthServiceInterface
	Logger      service.LoggerInterface
	Email       service.EmailInterface
	EmailErrors chan error
}

func New(userService service.UserServiceInterface, charService service.CharacterServiceInterface, authService service.AuthServiceInterface, logService service.LoggerInterface, emailService service.EmailInterface) *UserHandler {
	return &UserHandler{
		User:        userService,
		Char:        charService,
		Auth:        authService,
		Logger:      logService,
		Email:       emailService,
		EmailErrors: make(chan error, 10),
	}
}

func (h *UserHandler) Register(ctx *fiber.Ctx) error {
	br := model.BaseResponse{
		Error:   true,
		Message: "A aparut o eroare interna.",
	}

	var registerData model.RegisterAPI

	if err := ctx.BodyParser(&registerData); err != nil {
		h.Logger.Exception(fmt.Sprintf("Register(): error parsing body request: %v", err))
		return ctx.Status(http.StatusUnprocessableEntity).JSON(br)
	}

	if err := registerData.Validate(); err != nil {
		h.Logger.Exception(fmt.Sprintf("Register(): error validating data to register: %v", err))
		br.Message = model.ErrorMsg(err.Error())
		return ctx.Status(http.StatusUnprocessableEntity).JSON(br)
	}

	fetched, err := h.User.Fetch(registerData.Username, registerData.Email)
	if err != nil {
		h.Logger.Exception(fmt.Sprintf("Register(): trying to duplicate register: %v", err))
		br.Message = "Exista deja un cont cu acest nume sau aceasta adresa de mail."
		return ctx.Status(fiber.StatusConflict).JSON(br)
	}

	if fetched {
		h.Logger.Exception("Register(): trying to duplicate register")
		br.Message = "Exista deja un cont cu acest nume sau aceasta adresa de mail."
		return ctx.Status(fiber.StatusConflict).JSON(br)
	}

	if err = h.User.Create(&registerData); err != nil {
		h.Logger.Exception(fmt.Sprintf("Register(): error creating account: %v", err))
		return ctx.Status(http.StatusInternalServerError).JSON(br)
	}

	timestamp := time.Now().Unix()
	token := service.GenerateToken(registerData.Email, timestamp)
	confirmationLink := fmt.Sprintf("https://app.ro/internal-ucp-api/v1/confirm?email=%s&token=%s&timestamp=%d", registerData.Email, token, timestamp)
	emailBody := fmt.Sprintf(service.ConfirmAccountEmail, registerData.Username, confirmationLink)

	go func() {
		if err = h.Email.SendEmail(registerData.Email, "Confirmare cont UCP", emailBody); err != nil {
			h.Logger.Exception("Register(): Failed to send confirmation email: " + err.Error())
		}

		select {
		case h.EmailErrors <- err:
		default:
			h.Logger.Exception("Register(): EmailErrors channel is full or unbuffered.")
		}
	}()

	if err = <-h.EmailErrors; err != nil {
		br.Message = "Nu a putut fi trimis mailul catre adresa oferita."
		return ctx.Status(http.StatusInternalServerError).JSON(br)
	}

	return ctx.Status(http.StatusCreated).JSON(model.BaseResponse{
		Error:   false,
		Message: "",
	})
}

func (h *UserHandler) Confirm(ctx *fiber.Ctx) error {
	br := model.BaseResponse{
		Error:   true,
		Message: "A aparut o eroare interna.",
	}

	email := ctx.Query("email")
	token := ctx.Query("token")
	timestampStr := ctx.Query("timestamp")

	if email == "" || token == "" || timestampStr == "" {
		br.Message = "Parametrii trebuie completati."
		return ctx.Status(http.StatusUnprocessableEntity).JSON(br)
	}

	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(br)
	}

	if !service.ValidateToken(email, token, timestamp) {
		br.Message = "Token-ul este incorect."
		return ctx.Status(fiber.StatusUnauthorized).JSON(br)
	}

	if time.Now().Unix()-timestamp > 15*60 {
		br.Message = "Token-ul a expirat."
		return ctx.Status(fiber.StatusUnauthorized).JSON(br)
	}

	if err = h.User.ActivateAccount(email); err != nil {
		h.Logger.Exception("Confirm(): failed to activate account: " + err.Error())
		br.Message = "Contul nu poate fi activat."
		return ctx.Status(fiber.StatusInternalServerError).JSON(br)
	}

	return ctx.Redirect("/", fiber.StatusFound)
}

func (h *UserHandler) Login(ctx *fiber.Ctx) error {
	br := model.BaseResponse{
		Error:   true,
		Message: "Numele sau parola sunt gresite.",
	}

	var loginData model.LoginAPI

	if err := ctx.BodyParser(&loginData); err != nil {
		h.Logger.Exception(fmt.Sprintf("Login(): error parsing body request: %v", err))
		return ctx.Status(http.StatusUnprocessableEntity).JSON(br)
	}

	if err := loginData.Validate(); err != nil {
		h.Logger.Exception(fmt.Sprintf("Login(): error validating data: %v", err))
		return ctx.Status(http.StatusUnprocessableEntity).JSON(br)
	}

	banned, err := h.User.CheckForBan(loginData.Username)
	if err != nil {
		h.Logger.Exception(fmt.Sprintf("Login(): error checking for ban for user %s: %v", loginData.Username, err))
		return ctx.Status(http.StatusOK).JSON(br)
	}

	if banned {
		br.Message = "You are banned."
		return ctx.Status(http.StatusOK).JSON(br)
	}

	fetched, err := h.User.Fetch(loginData.Username, "")
	if err != nil {
		h.Logger.Exception("Login(): user doesn't exist " + err.Error())
		return ctx.Status(fiber.StatusConflict).JSON(br)
	}

	if !fetched {
		return ctx.Status(fiber.StatusConflict).JSON(br)
	}

	activated, err := h.User.CheckActivation(loginData.Username)
	if err != nil {
		h.Logger.Exception("Login(): error checking for activation status:" + err.Error())
		br.Message = "Contul nu este activat. Verifica adresa de email."
		return ctx.Status(http.StatusConflict).JSON(br)
	}

	if !activated {
		br.Message = "Contul nu este activat. Verifica adresa de email."
		return ctx.Status(http.StatusConflict).JSON(br)
	}

	if err = h.User.Verify(&loginData); err != nil {
		h.Logger.Exception(fmt.Sprintf("Login(): error fetching account: %v", err))
		return ctx.Status(http.StatusUnauthorized).JSON(br)
	}

	isTester, errTester := h.User.IsTester(loginData.Username)
	if errTester != nil {
		h.Logger.Exception(fmt.Sprintf("Login(): error fetching account: %v", errTester))
	}

	isAdmin, errAdmin := h.User.IsAdmin(loginData.Username)
	if errAdmin != nil {
		h.Logger.Exception(fmt.Sprintf("Login(): error fetching account: %v", errAdmin))
		return ctx.SendStatus(http.StatusInternalServerError)
	}

	if err = h.Auth.SaveSession(ctx, loginData.Username, isTester, isAdmin); err != nil {
		h.Logger.Exception(fmt.Sprintf("Login(): error saving session: %v", err))
		return ctx.SendStatus(http.StatusInternalServerError)
	}

	return ctx.Status(http.StatusAccepted).JSON(model.BaseResponse{
		Error:   false,
		Message: "",
	})
}

func (h *UserHandler) Logout(ctx *fiber.Ctx) error {
	if err := h.Auth.DestroySession(ctx); err != nil {
		h.Logger.Exception(fmt.Sprintf("Logout() error logging out: %v", err))
		return ctx.SendStatus(http.StatusUnauthorized)
	}

	return ctx.SendStatus(http.StatusOK)
}

func (h *UserHandler) ResetRequest(ctx *fiber.Ctx) error {
	br := model.BaseResponse{
		Error:   true,
		Message: "A fost intampinata o eroare interna.",
	}

	email := ctx.Query("email")

	if _, err := mail.ParseAddress(email); err != nil {
		br.Message = "Adresa de email este invalida."
		return ctx.Status(http.StatusBadRequest).JSON(br)
	}

	timestamp := time.Now().Unix()
	token := service.GenerateToken(email, timestamp)

	confirmationLink := fmt.Sprintf("https://app.ro/internal-ucp-api/v1/confirm-reset?email=%s&token=%s&timestamp=%d", email, token, timestamp)
	emailBody := fmt.Sprintf(service.ResetPasswordEmail, confirmationLink)

	go func() {
		var err error
		if err = h.Email.SendEmail(email, "Confirmare cont UCP", emailBody); err != nil {
			h.Logger.Exception("Register(): Failed to send confirmation email: " + err.Error())
		}

		select {
		case h.EmailErrors <- err:
		default:
			h.Logger.Exception("Register(): EmailErrors channel is full or unbuffered.")
		}
	}()

	if err := <-h.EmailErrors; err != nil {
		br.Message = "Nu a putut fi trimis mailul catre adresa oferita."
		return ctx.Status(http.StatusInternalServerError).JSON(br)
	}

	return ctx.Status(http.StatusOK).JSON(model.BaseResponse{
		Error:   false,
		Message: "",
	})
}

func (h *UserHandler) ConfirmReset(ctx *fiber.Ctx) error {
	br := model.BaseResponse{
		Error:   true,
		Message: "A fost intampinata o eroare interna.",
	}

	resetPwd := model.UpdatePassword{
		Email:       ctx.Query("email"),
		Token:       ctx.Query("token"),
		Timestamp:   int64(ctx.QueryInt("timestamp")),
		NewPassword: "",
	}

	if _, err := mail.ParseAddress(resetPwd.Email); err != nil || resetPwd.Token == "" || resetPwd.Timestamp == 0 {
		return ctx.Status(http.StatusBadRequest).JSON(br)
	}

	ts := time.Now().Unix()
	if ts-resetPwd.Timestamp > 15*60 {
		br.Message = "Token-ul este expirat."
		return ctx.Status(http.StatusBadRequest).JSON(br)
	}

	if !service.ValidateToken(resetPwd.Email, resetPwd.Token, resetPwd.Timestamp) {
		br.Message = "Token-ul este invalid."
		return ctx.Status(http.StatusBadRequest).JSON(br)
	}

	url := fmt.Sprintf("/password-reset?email=%s&token=%s&timestamp=%d", resetPwd.Email, resetPwd.Token, resetPwd.Timestamp)
	return ctx.Status(http.StatusFound).Redirect(url)
}

func (h *UserHandler) UpdatePassword(ctx *fiber.Ctx) error {
	br := model.BaseResponse{
		Error:   true,
		Message: "A fost intampinata o eroare interna.",
	}

	var resetPwd model.UpdatePassword

	if err := ctx.BodyParser(&resetPwd); err != nil {
		return ctx.Status(http.StatusUnprocessableEntity).JSON(br)
	}

	if err := resetPwd.Validate(); err != nil {
		br.Message = model.ErrorMsg(err.Error())
		return ctx.Status(http.StatusUnprocessableEntity).JSON(br)
	}

	ts := time.Now().Unix()
	if ts-resetPwd.Timestamp > 15*60 {
		br.Message = "Token-ul este expirat."
		return ctx.Status(http.StatusBadRequest).JSON(br)
	}

	if !service.ValidateToken(resetPwd.Email, resetPwd.Token, resetPwd.Timestamp) {
		br.Message = "Token-ul este invalid."
		return ctx.Status(http.StatusBadRequest).JSON(br)
	}

	if err := h.User.UpdatePassword(resetPwd.Email, resetPwd.NewPassword); err != nil {
		h.Logger.Exception(fmt.Sprintf("ConfirmReset(): error updating new password: %v", err))
		return ctx.Status(http.StatusInternalServerError).JSON(br)
	}

	return ctx.Status(http.StatusOK).JSON(model.BaseResponse{
		Error:   false,
		Message: "",
	})
}

func (h *UserHandler) CheckAuth(ctx *fiber.Ctx) error {
	br := model.BaseResponse{
		Error:   true,
		Message: "you are not authenticated",
	}

	name, _, _, err := h.Auth.CheckSession(ctx)
	if err != nil {
		h.Logger.Exception(fmt.Sprintf("CheckAuth(): error checking for session: %v", err))
		return ctx.Status(http.StatusInternalServerError).JSON(br)
	}

	if name == "" {
		h.Logger.Exception("CheckAuth(): session doesn't exist: user is not logged in")
		return ctx.Status(http.StatusOK).JSON(br)
	}

	return ctx.Status(http.StatusOK).JSON(fiber.Map{
		"authenticated": true,
		"user":          name,
	})
}

func (h *UserHandler) GetStats(ctx *fiber.Ctx) error {
	name, _, _, err := h.Auth.CheckSession(ctx)
	if err != nil {
		h.Logger.Exception(fmt.Sprintf("GetStats(): error checking for session: %v", err))
		return ctx.SendStatus(http.StatusInternalServerError)
	}

	if name == "" {
		h.Logger.Exception("GetStats(): session doesn't exist: user is not logged in")
		return ctx.SendStatus(http.StatusUnauthorized)
	}

	data, err := h.User.GetStats(name)
	if err != nil {
		h.Logger.Exception(fmt.Sprintf("GetStats() error fetching stats: %v", err))
		return ctx.SendStatus(http.StatusInternalServerError)
	}

	return ctx.Status(http.StatusOK).JSON(data)
}

func (h *UserHandler) GetStaff(ctx *fiber.Ctx) error {
	br := model.BaseResponse{
		Error:   true,
		Message: "can't get data",
	}

	data, err := h.User.GetStaff()
	if err != nil {
		h.Logger.Exception(fmt.Sprintf("GetStats() error fetching stats: %v", err))
		return ctx.Status(http.StatusInternalServerError).JSON(br)
	}

	type response struct {
		model.BaseResponse
		Data []model.GetStaffAPI `json:"data"`
	}

	return ctx.Status(fiber.StatusOK).JSON(response{
		BaseResponse: model.BaseResponse{},
		Data:         data,
	})
}

func (h *UserHandler) ServerStats(ctx *fiber.Ctx) error {
	br := model.BaseResponse{
		Error:   true,
		Message: "can't get data",
	}

	data, err := h.User.GetServerStats()
	if err != nil {
		h.Logger.Exception(fmt.Sprintf("ServerStats() error fetching stats: %v", err))
		return ctx.Status(http.StatusInternalServerError).JSON(br)
	}

	type response struct {
		model.BaseResponse
		Data model.ServerStatsAPI `json:"data"`
	}

	return ctx.Status(fiber.StatusOK).JSON(response{
		BaseResponse: model.BaseResponse{},
		Data:         *data,
	})
}

func (h *UserHandler) CheckAdmin(ctx *fiber.Ctx) error {
	br := model.BaseResponse{
		Error:   true,
		Message: "you are not authenticated",
	}

	name, admin, tester, err := h.Auth.CheckSession(ctx)
	if err != nil {
		h.Logger.Exception(fmt.Sprintf("CheckAdmin(): error checking for admin session: %v", err))
		return ctx.Status(http.StatusInternalServerError).JSON(br)
	}

	if name == "" {
		h.Logger.Exception("CheckAdmin(): invalid session: user is not logged in ")
		return ctx.Status(http.StatusOK).JSON(br)
	}

	if !admin && !tester {
		h.Logger.Exception("CheckAdmin(): invalid session: user doesn't have admin rights")
		return ctx.Status(http.StatusOK).JSON(br)
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"user":      name,
		"is_tester": tester,
		"is_admin":  admin,
	})
}

func (h *UserHandler) CreateCharacter(ctx *fiber.Ctx) error {
	br := model.BaseResponse{
		Error:   true,
		Message: "A fost intampinata o eroare interna.",
	}

	name, _, _, err := h.Auth.CheckSession(ctx)
	if err != nil {
		h.Logger.Exception(fmt.Sprintf("CreateCharacter(): error checking for session: %v", err))
		return ctx.Status(http.StatusInternalServerError).JSON(br)
	}

	if name == "" {
		h.Logger.Exception("CreateCharacter(): session doesn't exist: user is not logged in")
		return ctx.Status(http.StatusUnauthorized).JSON(br)
	}

	var createChar model.CharacterDataAPI
	if err = ctx.BodyParser(&createChar); err != nil {
		h.Logger.Exception(fmt.Sprintf("CreateCharacter(): error parsing body request: %v", err))
		return ctx.Status(http.StatusUnprocessableEntity).JSON(br)
	}

	stats, err := h.User.GetStats(name)
	if err != nil {
		h.Logger.Exception(fmt.Sprintf("CreateCharacter(): can't get stats: %v", err))
		return ctx.Status(http.StatusInternalServerError).JSON(br)
	}

	if stats != nil {
		if stats.Characters >= 5 {
			h.Logger.Exception("CreateCharacter(): can't create more characters")
			br.Message = "Ai atins numarul maxim de caractere."
			return ctx.Status(http.StatusConflict).JSON(br)
		}
	}

	if err = createChar.Validate(); err != nil {
		h.Logger.Exception("CreateCharacter(): error validating character data")
		br.Message = model.ErrorMsg(err.Error())
		return ctx.Status(http.StatusUnprocessableEntity).JSON(br)
	}

	createChar.Username = name
	if err = h.Char.Create(&createChar); err != nil {
		h.Logger.Exception(fmt.Sprintf("CreateCharacter(): error creating character: %v", err))
		br.Message = "Un caracter a fost deja creat cu acest nume."
		return ctx.Status(http.StatusInternalServerError).JSON(br)
	}

	return ctx.Status(http.StatusCreated).JSON(model.BaseResponse{
		Error:   false,
		Message: "",
	})
}

func (h *UserHandler) WaitingList(ctx *fiber.Ctx) error {
	name, isAdmin, isTester, err := h.Auth.CheckSession(ctx)
	if err != nil {
		h.Logger.Exception(fmt.Sprintf("WaitingList(): error checking for session: %v", err))
		return ctx.SendStatus(http.StatusInternalServerError)
	}

	if name == "" {
		h.Logger.Exception("WaitingList(): session doesn't exist: user is not logged in")
		return ctx.SendStatus(http.StatusUnauthorized)
	}

	if !isAdmin && !isTester {
		h.Logger.Exception(fmt.Sprintf("WaitingList(): user %s doesn't have admin rights", name))
		return ctx.SendStatus(http.StatusUnauthorized)
	}

	list, err := h.Char.FetchWaiting()
	if err != nil {
		h.Logger.Exception("WaitingList(): session doesn't exist: user is not logged in")
		return ctx.SendStatus(http.StatusUnauthorized)
	}

	return ctx.Status(http.StatusOK).JSON(list)
}

func (h *UserHandler) AcceptCharacter(ctx *fiber.Ctx) error {
	br := model.BaseResponse{
		Error:   true,
		Message: "Caracterul nu a putut fi acceptat.",
	}

	name, isAdmin, isTester, err := h.Auth.CheckSession(ctx)
	if err != nil {
		h.Logger.Exception(fmt.Sprintf("AcceptCharacter(): error checking for session: %v", err))
		return ctx.Status(http.StatusInternalServerError).JSON(br)
	}

	if name == "" {
		h.Logger.Exception("AcceptCharacter(): session doesn't exist: user is not logged in")
		return ctx.Status(http.StatusUnauthorized).JSON(br)
	}

	if !isAdmin && !isTester {
		h.Logger.Exception(fmt.Sprintf("AcceptCharacter(): user %s doesn't have admin rights", name))
		return ctx.Status(http.StatusUnauthorized).JSON(br)
	}

	var acceptChar model.CharacterAPI
	if err = ctx.BodyParser(&acceptChar); err != nil {
		h.Logger.Exception(fmt.Sprintf("AcceptCharacter(): error parsing body request: %v", err))
		return ctx.Status(http.StatusUnprocessableEntity).JSON(br)
	}

	acceptChar.AcceptedBy = name

	if err = h.Char.AcceptCharacter(acceptChar); err != nil {
		h.Logger.Exception(fmt.Sprintf("AcceptCharacter(): can't accept character: %v", err))
		return ctx.Status(http.StatusInternalServerError).JSON(br)
	}

	email, err := h.User.FetchMail(acceptChar.Username)
	if err != nil {
		h.Logger.Exception(fmt.Sprintf("AcceptCharacter(): can't get email for character: %v", err))
		return ctx.Status(http.StatusInternalServerError).JSON(br)
	}

	emailBody := fmt.Sprintf(service.AcceptCharacterEmail, acceptChar.Username, acceptChar.CharacterName, time.Now().Format("02/01/2006, 15:04"))

	go func() {
		if err = h.Email.SendEmail(email, "SA-RP: Caracter acceptat", emailBody); err != nil {
			h.Logger.Exception(fmt.Sprintf("AcceptCharacter(): can't send email: %v", err))
		}
	}()

	return ctx.Status(http.StatusOK).JSON(model.BaseResponse{
		Error:   false,
		Message: "",
	})
}

func (h *UserHandler) RejectCharacter(ctx *fiber.Ctx) error {
	br := model.BaseResponse{
		Error:   true,
		Message: "Caracterul nu a putut fi refuzat",
	}

	name, isAdmin, isTester, err := h.Auth.CheckSession(ctx)
	if err != nil {
		h.Logger.Exception(fmt.Sprintf("RejectCharacter(): error checking for session: %v", err))
		return ctx.Status(http.StatusInternalServerError).JSON(br)
	}

	if name == "" {
		h.Logger.Exception("RejectCharacter(): session doesn't exist: user is not logged in")
		return ctx.Status(http.StatusUnauthorized).JSON(br)
	}

	if !isAdmin && !isTester {
		h.Logger.Exception(fmt.Sprintf("RejectCharacter(): user %s doesn't have admin rights", name))
		return ctx.Status(http.StatusUnauthorized).JSON(br)
	}

	var declineChar model.RejectCharacterAPI
	if err = ctx.BodyParser(&declineChar); err != nil {
		h.Logger.Exception(fmt.Sprintf("RejectCharacter(): error parsing body request: %v", err))
		return ctx.Status(http.StatusUnprocessableEntity).JSON(br)
	}

	if err = h.Char.DeclineCharacter(declineChar); err != nil {
		h.Logger.Exception(fmt.Sprintf("RejectCharacter(): can't accept character: %v", err))
		return ctx.Status(http.StatusInternalServerError).JSON(br)
	}

	email, err := h.User.FetchMail(declineChar.Username)
	if err != nil {
		h.Logger.Exception(fmt.Sprintf("RejectCharacter(): can't get email for character: %v", err))
		return ctx.Status(http.StatusInternalServerError).JSON(br)
	}

	emailBody := fmt.Sprintf(service.DeclineCharacterEmail, declineChar.Username, declineChar.CharacterName, time.Now().Format("02/01/2006, 15:04"), declineChar.Reason, name)

	go func() {
		if err = h.Email.SendEmail(email, "SA-RP: Caracter refuzat", emailBody); err != nil {
			h.Logger.Exception(fmt.Sprintf("RejectCharacter(): can't send email: %v", err))
			return
		}
	}()

	return ctx.Status(http.StatusOK).JSON(model.BaseResponse{
		Error:   false,
		Message: "",
	})
}

func (h *UserHandler) FetchCharacter(ctx *fiber.Ctx) error {
	br := model.BaseResponse{
		Error:   true,
		Message: "Caracterul nu a putut fi gasit.",
	}

	name, isAdmin, _, err := h.Auth.CheckSession(ctx)
	if err != nil {
		h.Logger.Exception(fmt.Sprintf("FetchCharacter(): error checking for session: %v", err))
		return ctx.Status(http.StatusInternalServerError).JSON(br)
	}

	if name == "" {
		h.Logger.Exception("FetchCharacter(): session doesn't exist: user is not logged in")
		return ctx.Status(http.StatusUnauthorized).JSON(br)
	}

	if !isAdmin {
		h.Logger.Exception(fmt.Sprintf("FetchCharacter(): user %s doesn't have admin rights", name))
		return ctx.Status(http.StatusUnauthorized).JSON(br)
	}

	var data model.CharacterAPI
	if err = ctx.BodyParser(&data); err != nil {
		h.Logger.Exception(fmt.Sprintf("FetchCharacter(): error parsing body request: %v", err))
		return ctx.Status(http.StatusUnprocessableEntity).JSON(br)
	}

	fetchData, errFetch := h.User.FetchCharacter(data.CharacterName)
	if errFetch != nil {
		h.Logger.Exception(fmt.Sprintf("FetchCharacter(): error fetching character: %v", errFetch))
		return ctx.Status(http.StatusNotFound).JSON(br)
	}

	type response struct {
		model.BaseResponse
		Data *model.CharacterAPI `json:"data"`
	}

	return ctx.Status(http.StatusOK).JSON(response{
		BaseResponse: model.BaseResponse{
			Error:   false,
			Message: "",
		},
		Data: fetchData,
	})
}

func (h *UserHandler) BanList(ctx *fiber.Ctx) error {
	name, isAdmin, _, err := h.Auth.CheckSession(ctx)
	if err != nil {
		h.Logger.Exception(fmt.Sprintf("BanList(): error checking for session: %v", err))
		return ctx.SendStatus(http.StatusInternalServerError)
	}

	if name == "" {
		h.Logger.Exception("BanList(): session doesn't exist: user is not logged in")
		return ctx.SendStatus(http.StatusUnauthorized)
	}

	if !isAdmin {
		h.Logger.Exception(fmt.Sprintf("BanList(): user %s doesn't have admin rights", name))
		return ctx.SendStatus(http.StatusUnauthorized)
	}

	bans, errBans := h.User.BanList()
	if errBans != nil {
		h.Logger.Exception(fmt.Sprintf("BanList(): error fetching character: %v", errBans))
		return ctx.SendStatus(http.StatusNotFound)
	}

	return ctx.Status(http.StatusOK).JSON(bans)
}

func (h *UserHandler) Ban(ctx *fiber.Ctx) error {
	br := model.BaseResponse{
		Error:   true,
		Message: "Jucatorul nu a putut fi banat.",
	}
	name, isAdmin, _, err := h.Auth.CheckSession(ctx)
	if err != nil {
		h.Logger.Exception(fmt.Sprintf("Ban(): error checking for session: %v", err))
		return ctx.Status(http.StatusInternalServerError).JSON(br)
	}

	if name == "" {
		h.Logger.Exception("Ban(): session doesn't exist: user is not logged in")
		return ctx.Status(http.StatusUnauthorized).JSON(br)
	}

	if !isAdmin {
		h.Logger.Exception(fmt.Sprintf("Ban(): user %s doesn't have admin rights", name))
		return ctx.Status(http.StatusUnauthorized).JSON(br)
	}

	var data model.BanAPI
	if err = ctx.BodyParser(&data); err != nil {
		h.Logger.Exception(fmt.Sprintf("Ban(): error parsing body request: %v", err))
		return ctx.Status(http.StatusUnprocessableEntity).JSON(br)
	}

	data.AdminName = name

	if errBan := h.User.Ban(&data); errBan != nil {
		h.Logger.Exception(fmt.Sprintf("Ban(): error fetching character: %v", errBan))
		return ctx.Status(http.StatusNotFound).JSON(br)
	}

	return ctx.Status(http.StatusOK).JSON(model.BaseResponse{
		Error:   false,
		Message: "",
	})
}

func (h *UserHandler) Unban(ctx *fiber.Ctx) error {
	br := model.BaseResponse{
		Error:   true,
		Message: "Jucatorul nu a putut fi debanat.",
	}

	name, isAdmin, _, err := h.Auth.CheckSession(ctx)
	if err != nil {
		h.Logger.Exception(fmt.Sprintf("Unban(): error checking for session: %v", err))
		return ctx.Status(http.StatusInternalServerError).JSON(br)
	}

	if name == "" {
		h.Logger.Exception("Unban(): session doesn't exist: user is not logged in")
		return ctx.Status(http.StatusUnauthorized).JSON(br)
	}

	if !isAdmin {
		h.Logger.Exception(fmt.Sprintf("Unban(): user %s doesn't have admin rights", name))
		return ctx.Status(http.StatusUnauthorized).JSON(br)
	}

	var data model.BanAPI
	if err = ctx.BodyParser(&data); err != nil {
		h.Logger.Exception(fmt.Sprintf("Unban(): error parsing body request: %v", err))
		return ctx.Status(http.StatusUnprocessableEntity).JSON(br)
	}

	data.AdminName = name

	if data.Username == "" {
		h.Logger.Exception(fmt.Sprintf("Unban(): can't have name empty"))
		return ctx.Status(http.StatusUnprocessableEntity).JSON(br)
	}

	if errUnban := h.User.Unban(&data); errUnban != nil {
		h.Logger.Exception(fmt.Sprintf("Unban(): error fetching character: %v", errUnban))
		return ctx.Status(http.StatusNotFound).JSON(br)
	}

	return ctx.Status(http.StatusOK).JSON(model.BaseResponse{
		Error:   false,
		Message: "",
	})
}

func (h *UserHandler) Ajail(ctx *fiber.Ctx) error {
	br := model.BaseResponse{
		Error:   true,
		Message: "Jucatorul nu a putut fi sanctionat.",
	}

	name, _, _, err := h.Auth.CheckSession(ctx)
	if err != nil {
		h.Logger.Exception(fmt.Sprintf("Ajail(): error checking for session: %v", err))
		return ctx.Status(http.StatusInternalServerError).JSON(br)
	}

	var data model.AjailAPI
	if err = ctx.BodyParser(&data); err != nil {
		h.Logger.Exception(fmt.Sprintf("Ajail(): error parsing body request: %v", err))
		return ctx.Status(http.StatusUnprocessableEntity).JSON(br)
	}

	data.AdminName = name

	if data.Character == "" || data.Reason == "" || data.Time == 0 {
		h.Logger.Exception(fmt.Sprintf("Ajail(): can't have empty fields"))
		br.Message = "Unul sau mai multe campuri nu sunt completate."
		return ctx.Status(http.StatusUnprocessableEntity).JSON(br)
	}

	if errAjail := h.User.Ajail(&data); errAjail != nil {
		h.Logger.Exception(fmt.Sprintf("Ajail(): error fetching character: %v", errAjail))
		return ctx.Status(http.StatusNotFound).JSON(br)
	}

	return ctx.Status(http.StatusOK).JSON(model.BaseResponse{
		Error:   false,
		Message: "",
	})
}

func (h *UserHandler) Logs(ctx *fiber.Ctx) error {
	br := model.BaseResponse{
		Error:   true,
		Message: "Log-urile nu au putut fi obtinute.",
	}

	var data model.LogsAPI
	if err := ctx.BodyParser(&data); err != nil {
		h.Logger.Exception(fmt.Sprintf("Logs(): error parsing body request: %v", err))
		return ctx.Status(http.StatusUnprocessableEntity).JSON(br)
	}

	logs, err := h.User.Logs(&data)
	if err != nil {
		h.Logger.Exception(fmt.Sprintf("Logs(): error fetching logs: %v", err))
		return ctx.Status(http.StatusInternalServerError).JSON(br)
	}

	type response struct {
		model.BaseResponse
		Logs []map[string]interface{} `json:"logs"`
	}

	return ctx.Status(http.StatusOK).JSON(response{
		BaseResponse: model.BaseResponse{
			Error:   false,
			Message: "",
		},
		Logs: logs,
	})
}
