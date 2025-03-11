package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"net/http"
	"sarp_backend/model"
	"sarp_backend/service"
	"testing"
	"time"
)

func TestRegister(t *testing.T) {
	tests := []struct {
		name           string
		mockFunc       func(*service.MockAuthService, *service.MockEmailService, *service.MockLoggerService)
		data           *model.RegisterAPI
		expectedStatus int
		expectedBody   *model.BaseResponse
	}{
		{
			"Visitor registers new account",
			func(auth *service.MockAuthService, email *service.MockEmailService, log *service.MockLoggerService) {
				email.On("SendEmail", testEmail, "Confirmare cont UCP", mock.AnythingOfType("string")).Return(nil)
			},
			&model.RegisterAPI{
				Username: testUsername,
				Email:    testEmail,
				Password: testPassword,
			},
			http.StatusCreated,
			nil,
		},
		{
			"Invalid data is passed on register",
			func(auth *service.MockAuthService, email *service.MockEmailService, log *service.MockLoggerService) {
				log.On("Exception", mock.AnythingOfType("string")).Return()
			},
			&model.RegisterAPI{
				Username: "",
				Email:    "",
				Password: "",
			},
			http.StatusUnprocessableEntity,
			&model.BaseResponse{
				Error:   true,
				Message: "Parola trebuie sa aiba minim 8 caractere (litere, cifre si caractere speciale)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := testRepository(t)
			defer testCleanup(t, repo)

			auth := new(service.MockAuthService)
			log := new(service.MockLoggerService)
			email := new(service.MockEmailService)

			tt.mockFunc(auth, email, log)

			app := testServer(service.NewUserService(repo), auth, email, nil, log)
			resp := testSendRequest(t, app, http.MethodPost, "/register", tt.data)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Unexpected response HTTP status code for test: %s", tt.name)

			if tt.expectedStatus != http.StatusCreated {
				var respBody model.BaseResponse
				if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
					t.Fatalf("Error decoding response body: %v", err)
				}

				assert.Equal(t, tt.expectedBody, &respBody, "Unexpected response body for test: %s", tt.name)
			}
		})
	}
}

func TestConfirm(t *testing.T) {
	ts := time.Now().Unix()
	token := service.GenerateToken(testEmail, ts)

	tests := []struct {
		name           string
		mockFunc       func(*service.MockAuthService, *service.MockEmailService, *service.MockLoggerService)
		email          string
		token          string
		expectedStatus int
		expectedBody   *model.BaseResponse
	}{
		{
			"User confirms their registration",
			func(auth *service.MockAuthService, email *service.MockEmailService, log *service.MockLoggerService) {
				email.On("SendEmail", testEmail, "Confirmare cont UCP", mock.AnythingOfType("string")).Return(nil)
			},
			testEmail,
			token,
			http.StatusFound,
			nil,
		},
		{
			"Invalid email is passed to be confirmed",
			func(auth *service.MockAuthService, email *service.MockEmailService, log *service.MockLoggerService) {
				email.On("SendEmail", testEmail, "Confirmare cont UCP", mock.AnythingOfType("string")).Return(nil)
				log.On("Exception", mock.AnythingOfType("string")).Return()
			},
			"invalid",
			token,
			http.StatusUnauthorized,
			&model.BaseResponse{
				Error:   true,
				Message: "Token-ul este incorect.",
			},
		},
		{
			"Invalid token is passed to be confirmed",
			func(auth *service.MockAuthService, email *service.MockEmailService, log *service.MockLoggerService) {
				email.On("SendEmail", testEmail, "Confirmare cont UCP", mock.AnythingOfType("string")).Return(nil)
				log.On("Exception", mock.AnythingOfType("string")).Return()
			},
			testEmail,
			uuid.NewString(),
			http.StatusUnauthorized,
			&model.BaseResponse{
				Error:   true,
				Message: "Token-ul este incorect.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := testRepository(t)
			defer testCleanup(t, repo)

			auth := new(service.MockAuthService)
			log := new(service.MockLoggerService)
			email := new(service.MockEmailService)

			tt.mockFunc(auth, email, log)

			app := testServer(service.NewUserService(repo), auth, email, nil, log)
			registerAccount(t, app)

			target := fmt.Sprintf("/confirm?email=%s&token=%s&timestamp=%d", tt.email, tt.token, ts)
			resp := testSendRequest(t, app, http.MethodGet, target, nil)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Unexpected response HTTP status code for test: %s", tt.name)

			if resp.StatusCode != http.StatusFound {
				var respBody model.BaseResponse
				if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
					t.Fatalf("Error decoding response body: %v", err)
				}
				assert.Equal(t, tt.expectedBody, &respBody, "Unexpected response HTTP body for test: %s", tt.name)
			}
		})
	}
}

func TestLogin(t *testing.T) {
	tests := []struct {
		name           string
		mockFunc       func(auth *service.MockAuthService, email *service.MockEmailService, log *service.MockLoggerService)
		data           model.LoginAPI
		expectedStatus int
		expectedBody   model.BaseResponse
	}{
		{
			"Visitor successfully authenticates",
			func(auth *service.MockAuthService, email *service.MockEmailService, log *service.MockLoggerService) {
				auth.On("SaveSession", mock.Anything, testUsername, false, false).Return(nil)
				email.On("SendEmail", testEmail, "Confirmare cont UCP", mock.AnythingOfType("string")).Return(nil)
			},
			model.LoginAPI{
				Username: testUsername,
				Password: testPassword,
			},
			http.StatusAccepted,
			model.BaseResponse{
				Error:   false,
				Message: "",
			},
		},
		{
			"Invalid credentials are passed on login",
			func(auth *service.MockAuthService, email *service.MockEmailService, log *service.MockLoggerService) {
				email.On("SendEmail", testEmail, "Confirmare cont UCP", mock.AnythingOfType("string")).Return(nil)
				log.On("Exception", mock.AnythingOfType("string")).Return()
			},
			model.LoginAPI{
				Username: testUsername,
				Password: "invalid",
			},
			http.StatusUnauthorized,
			model.BaseResponse{
				Error:   true,
				Message: "Numele sau parola sunt gresite.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := testRepository(t)
			defer testCleanup(t, repo)

			auth := new(service.MockAuthService)
			email := new(service.MockEmailService)
			log := new(service.MockLoggerService)

			tt.mockFunc(auth, email, log)

			app := testServer(service.NewUserService(repo), auth, email, nil, log)
			registerAndConfirmAccount(t, app)

			resp := testSendRequest(t, app, http.MethodPost, "/login", tt.data)

			var responseBody model.BaseResponse
			if err := json.NewDecoder(resp.Body).Decode(&responseBody); err != nil {
				t.Fatalf("Error decoding response body for test %s: %v", tt.name, err)
			}

			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Unexpected response HTTP status code for test: %s", tt.name)
			assert.Equal(t, tt.expectedBody, responseBody, "Unexpected response body for test: %s", tt.name)
		})
	}
}

func TestLoginUnconfirmedAccount(t *testing.T) {
	repo := testRepository(t)
	defer testCleanup(t, repo)

	auth := new(service.MockAuthService)
	email := new(service.MockEmailService)
	log := new(service.MockLoggerService)

	email.On("SendEmail", testEmail, "Confirmare cont UCP", mock.AnythingOfType("string")).Return(nil)

	app := testServer(service.NewUserService(repo), auth, email, nil, log)
	registerAccount(t, app)

	resp := testSendRequest(t, app, http.MethodPost, "/login", model.LoginAPI{
		Username: testUsername,
		Password: testPassword,
	})

	expectedBody := model.BaseResponse{
		Error:   true,
		Message: "Contul nu este activat. Verifica adresa de email.",
	}

	var responseBody model.BaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&responseBody); err != nil {
		t.Fatalf("Error decoding response body for test %s: %v", t.Name(), err)
	}

	assert.Equal(t, http.StatusConflict, resp.StatusCode, "Unexpected response HTTP status code for test: %s", t.Name())
	assert.Equal(t, expectedBody, responseBody, "Unexpected response body for test: %s", t.Name())
}

func TestLogout(t *testing.T) {
	tests := []struct {
		name           string
		mockFunc       func(auth *service.MockAuthService, log *service.MockLoggerService)
		expectedStatus int
	}{
		{
			"Authenticated user logs out successfully",
			func(auth *service.MockAuthService, log *service.MockLoggerService) {
				auth.On("DestroySession", mock.Anything).Return(nil)
			},
			http.StatusOK,
		},
		{
			"Unauthenticated user fails to log out",
			func(auth *service.MockAuthService, log *service.MockLoggerService) {
				auth.On("DestroySession", mock.Anything).Return(errors.New("no active session"))
				log.On("Exception", mock.AnythingOfType("string")).Return()
			},
			http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := new(service.MockAuthService)
			log := new(service.MockLoggerService)

			tt.mockFunc(auth, log)
			app := testServer(nil, auth, nil, nil, log)
			resp := testSendRequest(t, app, http.MethodPost, "/logout", nil)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

func TestCheckAuth(t *testing.T) {
	tests := []struct {
		name           string
		mockFunc       func(*service.MockAuthService, *service.MockLoggerService)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Authenticated user is verified",
			mockFunc: func(auth *service.MockAuthService, logger *service.MockLoggerService) {
				auth.On("CheckSession", mock.Anything).Return(testUsername, false, false, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: fiber.Map{
				"authenticated": true,
				"user":          testUsername,
			},
		},
		{
			name: "User is not authenticated",
			mockFunc: func(auth *service.MockAuthService, logger *service.MockLoggerService) {
				auth.On("CheckSession", mock.Anything).Return("", false, false, nil)
				logger.On("Exception", mock.AnythingOfType("string")).Return()
			},
			expectedStatus: http.StatusOK,
			expectedBody:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := new(service.MockLoggerService)
			auth := new(service.MockAuthService)

			tt.mockFunc(auth, logger)

			app := testServer(nil, auth, nil, nil, logger)
			resp := testSendRequest(t, app, http.MethodGet, "/check-auth", nil)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Unexpected status code for test: %s", tt.name)

			if tt.expectedBody != nil {
				var responseBody map[string]interface{}
				if err := json.NewDecoder(resp.Body).Decode(&responseBody); err != nil {
					t.Fatalf("Failed to parse JSON response: %v", err)
				}
				assert.Equal(t, tt.expectedBody, responseBody, "Unexpected response body for test: %s", tt.name)
			}
		})
	}
}

func TestGetStats(t *testing.T) {
	tests := []struct {
		name           string
		mockFunc       func(*service.MockAuthService, *service.MockEmailService, *service.MockLoggerService)
		expectedStatus int
		expectedBody   *model.GetStatsAPI
	}{
		{
			"User is authenticated and gets their stats retrieved",
			func(auth *service.MockAuthService, email *service.MockEmailService, logger *service.MockLoggerService) {
				auth.On("CheckSession", mock.Anything).Return(testUsername, false, false, nil).Once()
				email.On("SendEmail", testEmail, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
			},
			http.StatusOK,
			&model.GetStatsAPI{
				Username:      testUsername,
				Admin:         0,
				Tester:        0,
				DonateRank:    0,
				Characters:    0,
				LastLogin:     0,
				CharacterList: nil,
			},
		},
		{
			"User is not authenticated",
			func(auth *service.MockAuthService, email *service.MockEmailService, logger *service.MockLoggerService) {
				auth.On("CheckSession", mock.Anything).Return("", false, false, nil)
				logger.On("Exception", mock.AnythingOfType("string")).Return()
				email.On("SendEmail", testEmail, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
			},
			http.StatusUnauthorized,
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := testRepository(t)
			defer testCleanup(t, repo)

			auth := new(service.MockAuthService)
			email := new(service.MockEmailService)
			logger := new(service.MockLoggerService)

			tt.mockFunc(auth, email, logger)

			app := testServer(service.NewUserService(repo), auth, email, nil, logger)

			registerAndConfirmAccount(t, app)
			resp := testSendRequest(t, app, http.MethodGet, "/get-data", nil)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Unexpected status code for test: %s", tt.name)

			if tt.expectedBody != nil {
				var respBody model.GetStatsAPI
				if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
					t.Fatalf("Error decoding response body: %v", err)
				}

				assert.Equal(t, tt.expectedBody, &respBody, "Unexpected response body for tests: %s", tt.name)
			}
		})
	}
}

func TestGetStaff(t *testing.T) {
	tests := []struct {
		name           string
		mockFunc       func(*service.MockAuthService, *service.MockEmailService, *service.MockLoggerService)
		expectedStatus int
		expectedBody   *model.GetStaffAPI
	}{
		{
			"User is authenticated and gets staff list",
			func(auth *service.MockAuthService, email *service.MockEmailService, logger *service.MockLoggerService) {
				email.On("SendEmail", testEmail, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
			},
			http.StatusOK,
			&model.GetStaffAPI{
				Username: "",
				Role:     "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := testRepository(t)
			defer testCleanup(t, repo)

			auth := new(service.MockAuthService)
			email := new(service.MockEmailService)
			logger := new(service.MockLoggerService)

			tt.mockFunc(auth, email, logger)

			app := testServer(service.NewUserService(repo), auth, email, nil, logger)

			registerAndConfirmAccount(t, app)
			resp := testSendRequest(t, app, http.MethodGet, "/get-staff", nil)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Unexpected status code for test: %s", tt.name)

			if tt.expectedBody != nil {
				var respBody model.GetStaffAPI
				if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
					t.Fatalf("Error decoding response body: %v", err)
				}

				assert.Equal(t, tt.expectedBody, &respBody, "Unexpected response body for tests: %s", tt.name)
			}
		})
	}
}

func TestServerStats(t *testing.T) {
	type response struct {
		Response model.BaseResponse
		Data     model.ServerStatsAPI `json:"data"`
	}

	tests := []struct {
		name           string
		mockFunc       func(*service.MockAuthService, *service.MockEmailService, *service.MockLoggerService)
		expectedStatus int
		expectedBody   *response
	}{
		{
			"User is authenticated and gets server stats",
			func(auth *service.MockAuthService, email *service.MockEmailService, logger *service.MockLoggerService) {
				email.On("SendEmail", testEmail, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
			},
			http.StatusOK,
			&response{
				model.BaseResponse{},
				model.ServerStatsAPI{
					Accounts: 1,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := testRepository(t)
			defer testCleanup(t, repo)

			auth := new(service.MockAuthService)
			email := new(service.MockEmailService)
			logger := new(service.MockLoggerService)

			tt.mockFunc(auth, email, logger)

			app := testServer(service.NewUserService(repo), auth, email, nil, logger)

			registerAndConfirmAccount(t, app)
			resp := testSendRequest(t, app, http.MethodGet, "/server-stats", nil)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Unexpected status code for test: %s", tt.name)

			if tt.expectedBody != nil {
				var respBody response
				if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
					t.Fatalf("Error decoding response body: %v", err)
				}

				assert.Equal(t, tt.expectedBody, &respBody, "Unexpected response body for tests: %s", tt.name)
			}
		})
	}
}

func TestCreateCharacter(t *testing.T) {
	tests := []struct {
		name           string
		mockFunc       func(*service.MockAuthService, *service.MockEmailService, *service.MockLoggerService)
		data           *model.CharacterDataAPI
		expectedStatus int
		expectedBody   *model.BaseResponse
	}{
		{
			"User is authenticated and sends correct data",
			func(auth *service.MockAuthService, email *service.MockEmailService, logger *service.MockLoggerService) {
				auth.On("CheckSession", mock.Anything).Return(testUsername, false, false, nil).Once()
				email.On("SendEmail", testEmail, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
			},
			&model.CharacterDataAPI{
				Username:        testUsername,
				CharacterName:   "Test_Test",
				CharacterAge:    18,
				CharacterGender: 0,
				CharacterOrigin: "test",
			},
			http.StatusCreated,
			&model.BaseResponse{},
		},
		{
			"User is not authenticated",
			func(auth *service.MockAuthService, email *service.MockEmailService, logger *service.MockLoggerService) {
				auth.On("CheckSession", mock.Anything).Return("", false, false, nil)
				logger.On("Exception", mock.AnythingOfType("string")).Return()
				email.On("SendEmail", testEmail, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
			},
			nil,
			http.StatusUnauthorized,
			nil,
		},
		{
			"User provides invalid character data",
			func(auth *service.MockAuthService, email *service.MockEmailService, logger *service.MockLoggerService) {
				auth.On("CheckSession", mock.Anything).Return(testUsername, false, false, nil).Once()
				logger.On("Exception", mock.AnythingOfType("string")).Return()
				email.On("SendEmail", testEmail, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
			},
			&model.CharacterDataAPI{
				Username:        testUsername,
				CharacterName:   "",
				CharacterAge:    18,
				CharacterGender: 0,
				CharacterOrigin: "test",
			},
			http.StatusUnprocessableEntity,
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := testRepository(t)
			defer testCleanup(t, repo)

			auth := new(service.MockAuthService)
			email := new(service.MockEmailService)
			logger := new(service.MockLoggerService)

			tt.mockFunc(auth, email, logger)

			app := testServer(service.NewUserService(repo), auth, email, service.NewCharacterService(repo), logger)

			registerAndConfirmAccount(t, app)
			resp := testSendRequest(t, app, http.MethodPost, "/create-character", tt.data)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Unexpected status code for test: %s", tt.name)

			if tt.expectedBody != nil {
				var respBody model.BaseResponse
				if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
					t.Fatalf("Error decoding response body: %v", err)
				}

				assert.Equal(t, tt.expectedBody, &respBody, "Unexpected response body for tests: %s", tt.name)
			}
		})
	}
}

func TestCheckAdmin(t *testing.T) {
	tests := []struct {
		name           string
		mockFunc       func(*service.MockAuthService, *service.MockEmailService, *service.MockLoggerService)
		expectedStatus int
	}{
		{
			name: "User is authenticated and with correct privilege",
			mockFunc: func(auth *service.MockAuthService, email *service.MockEmailService, logger *service.MockLoggerService) {
				auth.On("CheckSession", mock.Anything).Return(testUsername, true, false, nil).Once()
				email.On("SendEmail", testEmail, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			"User is authenticated but with wrong privilege",
			func(auth *service.MockAuthService, email *service.MockEmailService, logger *service.MockLoggerService) {
				auth.On("CheckSession", mock.Anything).Return(testUsername, false, false, nil).Once()
				logger.On("Exception", mock.AnythingOfType("string")).Return()
				email.On("SendEmail", testEmail, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
			},
			http.StatusOK,
		},
		{
			"User is not authenticated",
			func(auth *service.MockAuthService, email *service.MockEmailService, logger *service.MockLoggerService) {
				auth.On("CheckSession", mock.Anything).Return("", false, false, nil).Once()
				logger.On("Exception", mock.AnythingOfType("string")).Return()
				email.On("SendEmail", testEmail, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
			},
			http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := testRepository(t)
			defer testCleanup(t, repo)

			auth := new(service.MockAuthService)
			email := new(service.MockEmailService)
			logger := new(service.MockLoggerService)

			tt.mockFunc(auth, email, logger)

			app := testServer(service.NewUserService(repo), auth, email, nil, logger)

			registerAndConfirmAccount(t, app)
			resp := testSendRequest(t, app, http.MethodGet, "/restricted/check", nil)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Unexpected status code for test: %s", tt.name)
		})
	}
}

func TestWaitingList(t *testing.T) {
	tests := []struct {
		name           string
		mockFunc       func(*service.MockAuthService, *service.MockEmailService, *service.MockLoggerService)
		expectedStatus int
		expectedBody   []model.CharacterDataAPI
	}{
		{
			"User is authenticated and with correct privilege",
			func(auth *service.MockAuthService, email *service.MockEmailService, logger *service.MockLoggerService) {
				auth.On("CheckSession", mock.Anything).Return(testUsername, false, false, nil).Once()
				auth.On("CheckSession", mock.Anything).Return(testUsername, true, false, nil).Once()
				email.On("SendEmail", testEmail, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
			},
			http.StatusOK,
			[]model.CharacterDataAPI{
				{
					Username:        testUsername,
					CharacterName:   "Test_Test",
					CharacterAge:    18,
					CharacterGender: 0,
					CharacterOrigin: "test",
				},
			},
		},
		{
			"User is authenticated but with wrong privilege",
			func(auth *service.MockAuthService, email *service.MockEmailService, logger *service.MockLoggerService) {
				auth.On("CheckSession", mock.Anything).Return(testUsername, false, false, nil).Once()
				auth.On("CheckSession", mock.Anything).Return(testUsername, false, false, nil).Once()
				logger.On("Exception", mock.AnythingOfType("string")).Return()
				email.On("SendEmail", testEmail, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
			},
			http.StatusUnauthorized,
			nil,
		},
		{
			"User is not authenticated",
			func(auth *service.MockAuthService, email *service.MockEmailService, logger *service.MockLoggerService) {
				auth.On("CheckSession", mock.Anything).Return(testUsername, false, false, nil).Once()
				auth.On("CheckSession", mock.Anything).Return("", false, false, nil).Once()
				logger.On("Exception", mock.AnythingOfType("string")).Return()
				email.On("SendEmail", testEmail, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
			},
			http.StatusUnauthorized,
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := testRepository(t)
			defer testCleanup(t, repo)

			auth := new(service.MockAuthService)
			email := new(service.MockEmailService)
			logger := new(service.MockLoggerService)

			tt.mockFunc(auth, email, logger)

			app := testServer(service.NewUserService(repo), auth, email, service.NewCharacterService(repo), logger)

			registerAndConfirmAccount(t, app)
			createCharacter(t, app)
			resp := testSendRequest(t, app, http.MethodGet, "/restricted/waiting-list", nil)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Unexpected status code for test: %s", tt.name)

			if tt.expectedBody != nil {
				var respBody []model.CharacterDataAPI
				if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
					t.Fatalf("Error decoding response body: %v", err)
				}
				assert.Equal(t, tt.expectedBody, respBody, "Unexpected response body for test: %s", tt.name)
			}
		})
	}
}

func TestAcceptCharacter(t *testing.T) {
	tests := []struct {
		name           string
		mockFunc       func(*service.MockAuthService, *service.MockEmailService, *service.MockLoggerService)
		data           *model.CharacterAPI
		expectedStatus int
	}{
		{
			"User is authenticated and with correct privilege",
			func(auth *service.MockAuthService, email *service.MockEmailService, logger *service.MockLoggerService) {
				auth.On("CheckSession", mock.Anything).Return(testUsername, false, false, nil).Once()
				auth.On("CheckSession", mock.Anything).Return(testUsername, true, false, nil).Once()
				email.On("SendEmail", testEmail, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
			},
			&model.CharacterAPI{
				Username:      testUsername,
				CharacterName: "Test_Test",
				AcceptedBy:    testUsername,
			},
			http.StatusOK,
		},
		{
			"User is authenticated but with wrong privilege",
			func(auth *service.MockAuthService, email *service.MockEmailService, logger *service.MockLoggerService) {
				auth.On("CheckSession", mock.Anything).Return(testUsername, false, false, nil).Once()
				auth.On("CheckSession", mock.Anything).Return(testUsername, false, false, nil).Once()
				logger.On("Exception", mock.AnythingOfType("string")).Return()
				email.On("SendEmail", testEmail, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
			},
			&model.CharacterAPI{
				Username:      testUsername,
				CharacterName: "Test_Test",
				AcceptedBy:    testUsername,
			},
			http.StatusUnauthorized,
		},
		{
			"User is not authenticated",
			func(auth *service.MockAuthService, email *service.MockEmailService, logger *service.MockLoggerService) {
				auth.On("CheckSession", mock.Anything).Return(testUsername, false, false, nil).Once()
				auth.On("CheckSession", mock.Anything).Return("", false, false, nil).Once()
				logger.On("Exception", mock.AnythingOfType("string")).Return()
				email.On("SendEmail", testEmail, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
			},
			&model.CharacterAPI{
				Username:      testUsername,
				CharacterName: "Test_Test",
				AcceptedBy:    testUsername,
			},
			http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := testRepository(t)
			defer testCleanup(t, repo)

			auth := new(service.MockAuthService)
			email := new(service.MockEmailService)
			logger := new(service.MockLoggerService)

			tt.mockFunc(auth, email, logger)

			app := testServer(service.NewUserService(repo), auth, email, service.NewCharacterService(repo), logger)

			registerAndConfirmAccount(t, app)
			createCharacter(t, app)
			resp := testSendRequest(t, app, http.MethodPost, "/restricted/accept-character", tt.data)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Unexpected status code for test: %s", tt.name)
		})
	}
}

func TestRejectCharacter(t *testing.T) {
	tests := []struct {
		name           string
		mockFunc       func(*service.MockAuthService, *service.MockEmailService, *service.MockLoggerService)
		data           *model.CharacterAPI
		expectedStatus int
	}{
		{
			"User is authenticated and with correct privilege",
			func(auth *service.MockAuthService, email *service.MockEmailService, logger *service.MockLoggerService) {
				auth.On("CheckSession", mock.Anything).Return(testUsername, false, false, nil).Once()
				auth.On("CheckSession", mock.Anything).Return(testUsername, true, false, nil).Once()
				email.On("SendEmail", testEmail, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
			},
			&model.CharacterAPI{
				Username:      testUsername,
				CharacterName: "Test_Test",
				AcceptedBy:    testUsername,
			},
			http.StatusOK,
		},
		{
			"User is authenticated but with wrong privilege",
			func(auth *service.MockAuthService, email *service.MockEmailService, logger *service.MockLoggerService) {
				auth.On("CheckSession", mock.Anything).Return(testUsername, false, false, nil).Once()
				auth.On("CheckSession", mock.Anything).Return(testUsername, false, false, nil).Once()
				logger.On("Exception", mock.AnythingOfType("string")).Return()
				email.On("SendEmail", testEmail, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
			},
			&model.CharacterAPI{
				Username:      testUsername,
				CharacterName: "Test_Test",
				AcceptedBy:    testUsername,
			},
			http.StatusUnauthorized,
		},
		{
			"User is not authenticated",
			func(auth *service.MockAuthService, email *service.MockEmailService, logger *service.MockLoggerService) {
				auth.On("CheckSession", mock.Anything).Return(testUsername, false, false, nil).Once()
				auth.On("CheckSession", mock.Anything).Return("", false, false, nil).Once()
				logger.On("Exception", mock.AnythingOfType("string")).Return()
				email.On("SendEmail", testEmail, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
			},
			&model.CharacterAPI{
				Username:      testUsername,
				CharacterName: "Test_Test",
				AcceptedBy:    testUsername,
			},
			http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := testRepository(t)
			defer testCleanup(t, repo)

			auth := new(service.MockAuthService)
			email := new(service.MockEmailService)
			logger := new(service.MockLoggerService)

			tt.mockFunc(auth, email, logger)

			app := testServer(service.NewUserService(repo), auth, email, service.NewCharacterService(repo), logger)

			registerAndConfirmAccount(t, app)
			createCharacter(t, app)
			resp := testSendRequest(t, app, http.MethodPost, "/restricted/reject-character", tt.data)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Unexpected status code for test: %s", tt.name)
		})
	}
}

func TestFetchCharacter(t *testing.T) {
	tests := []struct {
		name           string
		mockFunc       func(*service.MockAuthService, *service.MockEmailService, *service.MockLoggerService)
		data           *model.CharacterAPI
		expectedStatus int
	}{
		{
			"User is authenticated and with correct privilege",
			func(auth *service.MockAuthService, email *service.MockEmailService, logger *service.MockLoggerService) {
				auth.On("CheckSession", mock.Anything).Return(testUsername, false, false, nil).Once()
				auth.On("CheckSession", mock.Anything).Return(testUsername, true, false, nil).Once()
				email.On("SendEmail", testEmail, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
			},
			&model.CharacterAPI{
				Username:      testUsername,
				CharacterName: "Test_Test",
				AcceptedBy:    testUsername,
			},
			http.StatusOK,
		},
		{
			"User is authenticated but with wrong privilege",
			func(auth *service.MockAuthService, email *service.MockEmailService, logger *service.MockLoggerService) {
				auth.On("CheckSession", mock.Anything).Return(testUsername, false, false, nil).Once()
				auth.On("CheckSession", mock.Anything).Return(testUsername, false, false, nil).Once()
				logger.On("Exception", mock.AnythingOfType("string")).Return()
				email.On("SendEmail", testEmail, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
			},
			&model.CharacterAPI{
				Username:      testUsername,
				CharacterName: "Test_Test",
				AcceptedBy:    testUsername,
			},
			http.StatusUnauthorized,
		},
		{
			"User is not authenticated",
			func(auth *service.MockAuthService, email *service.MockEmailService, logger *service.MockLoggerService) {
				auth.On("CheckSession", mock.Anything).Return(testUsername, false, false, nil).Once()
				auth.On("CheckSession", mock.Anything).Return("", false, false, nil).Once()
				logger.On("Exception", mock.AnythingOfType("string")).Return()
				email.On("SendEmail", testEmail, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
			},
			&model.CharacterAPI{
				Username:      testUsername,
				CharacterName: "Test_Test",
				AcceptedBy:    testUsername,
			},
			http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := testRepository(t)
			defer testCleanup(t, repo)

			auth := new(service.MockAuthService)
			email := new(service.MockEmailService)
			logger := new(service.MockLoggerService)

			tt.mockFunc(auth, email, logger)

			app := testServer(service.NewUserService(repo), auth, email, service.NewCharacterService(repo), logger)

			registerAndConfirmAccount(t, app)
			createCharacter(t, app)
			resp := testSendRequest(t, app, http.MethodPost, "/restricted/fetch-character", tt.data)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Unexpected status code for test: %s", tt.name)
		})
	}
}

func TestBanList(t *testing.T) {
	tests := []struct {
		name           string
		mockFunc       func(*service.MockAuthService, *service.MockEmailService, *service.MockLoggerService)
		expectedStatus int
	}{
		{
			"User is authenticated and with correct privilege",
			func(auth *service.MockAuthService, email *service.MockEmailService, logger *service.MockLoggerService) {
				auth.On("CheckSession", mock.Anything).Return(testUsername, false, false, nil).Once()
				auth.On("CheckSession", mock.Anything).Return(testUsername, true, false, nil).Once()
				email.On("SendEmail", testEmail, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
			},
			http.StatusOK,
		},
		{
			"User is authenticated but with wrong privilege",
			func(auth *service.MockAuthService, email *service.MockEmailService, logger *service.MockLoggerService) {
				auth.On("CheckSession", mock.Anything).Return(testUsername, false, false, nil).Once()
				auth.On("CheckSession", mock.Anything).Return(testUsername, false, false, nil).Once()
				logger.On("Exception", mock.AnythingOfType("string")).Return()
				email.On("SendEmail", testEmail, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
			},
			http.StatusUnauthorized,
		},
		{
			"User is not authenticated",
			func(auth *service.MockAuthService, email *service.MockEmailService, logger *service.MockLoggerService) {
				auth.On("CheckSession", mock.Anything).Return(testUsername, false, false, nil).Once()
				auth.On("CheckSession", mock.Anything).Return("", false, false, nil).Once()
				logger.On("Exception", mock.AnythingOfType("string")).Return()
				email.On("SendEmail", testEmail, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
			},
			http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := testRepository(t)
			defer testCleanup(t, repo)

			auth := new(service.MockAuthService)
			email := new(service.MockEmailService)
			logger := new(service.MockLoggerService)

			tt.mockFunc(auth, email, logger)

			app := testServer(service.NewUserService(repo), auth, email, service.NewCharacterService(repo), logger)

			registerAndConfirmAccount(t, app)
			createCharacter(t, app)
			resp := testSendRequest(t, app, http.MethodGet, "/restricted/ban-list", nil)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Unexpected status code for test: %s", tt.name)
		})
	}
}

func TestBan(t *testing.T) {
	tests := []struct {
		name           string
		mockFunc       func(*service.MockAuthService, *service.MockEmailService, *service.MockLoggerService)
		data           *model.BanAPI
		expectedStatus int
	}{
		{
			"User is authenticated and with correct privilege",
			func(auth *service.MockAuthService, email *service.MockEmailService, logger *service.MockLoggerService) {
				auth.On("CheckSession", mock.Anything).Return(testUsername, false, false, nil).Once()
				auth.On("CheckSession", mock.Anything).Return(testUsername, true, false, nil).Once()
				email.On("SendEmail", testEmail, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
			},
			&model.BanAPI{
				Username:   testUsername,
				Expire:     1,
				Reason:     "test",
				AdminName:  testUsername,
				Characters: nil,
			},
			http.StatusOK,
		},
		{
			"User is authenticated but with wrong privilege",
			func(auth *service.MockAuthService, email *service.MockEmailService, logger *service.MockLoggerService) {
				auth.On("CheckSession", mock.Anything).Return(testUsername, false, false, nil).Once()
				auth.On("CheckSession", mock.Anything).Return(testUsername, false, false, nil).Once()
				logger.On("Exception", mock.AnythingOfType("string")).Return()
				email.On("SendEmail", testEmail, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
			},
			&model.BanAPI{
				Username:   testUsername,
				Expire:     1,
				Reason:     "test",
				AdminName:  testUsername,
				Characters: nil,
			},
			http.StatusUnauthorized,
		},
		{
			"User is not authenticated",
			func(auth *service.MockAuthService, email *service.MockEmailService, logger *service.MockLoggerService) {
				auth.On("CheckSession", mock.Anything).Return(testUsername, false, false, nil).Once()
				auth.On("CheckSession", mock.Anything).Return("", false, false, nil).Once()
				logger.On("Exception", mock.AnythingOfType("string")).Return()
				email.On("SendEmail", testEmail, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
			},
			&model.BanAPI{
				Username:   testUsername,
				Expire:     1,
				Reason:     "test",
				AdminName:  testUsername,
				Characters: nil,
			},
			http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := testRepository(t)
			defer testCleanup(t, repo)

			auth := new(service.MockAuthService)
			email := new(service.MockEmailService)
			logger := new(service.MockLoggerService)

			tt.mockFunc(auth, email, logger)

			app := testServer(service.NewUserService(repo), auth, email, service.NewCharacterService(repo), logger)

			registerAndConfirmAccount(t, app)
			createCharacter(t, app)
			resp := testSendRequest(t, app, http.MethodPost, "/restricted/ban", tt.data)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Unexpected status code for test: %s", tt.name)
		})
	}
}

func TestUnban(t *testing.T) {
	tests := []struct {
		name           string
		mockFunc       func(*service.MockAuthService, *service.MockEmailService, *service.MockLoggerService)
		data           *model.BanAPI
		expectedStatus int
	}{
		{
			"User is authenticated and with correct privilege",
			func(auth *service.MockAuthService, email *service.MockEmailService, logger *service.MockLoggerService) {
				auth.On("CheckSession", mock.Anything).Return(testUsername, false, false, nil).Once()
				auth.On("CheckSession", mock.Anything).Return(testUsername, true, false, nil)
				email.On("SendEmail", testEmail, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
			},
			&model.BanAPI{
				Username:   testUsername,
				Expire:     1,
				Reason:     "test",
				AdminName:  testUsername,
				Characters: nil,
			},
			http.StatusOK,
		},
		{
			"User is authenticated but with wrong privilege",
			func(auth *service.MockAuthService, email *service.MockEmailService, logger *service.MockLoggerService) {
				auth.On("CheckSession", mock.Anything).Return(testUsername, false, false, nil).Once()
				auth.On("CheckSession", mock.Anything).Return(testUsername, true, false, nil).Once()
				auth.On("CheckSession", mock.Anything).Return(testUsername, false, false, nil).Once()
				logger.On("Exception", mock.AnythingOfType("string")).Return()
				email.On("SendEmail", testEmail, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
			},
			&model.BanAPI{
				Username:   testUsername,
				Expire:     1,
				Reason:     "test",
				AdminName:  testUsername,
				Characters: nil,
			},
			http.StatusUnauthorized,
		},
		{
			"User is not authenticated",
			func(auth *service.MockAuthService, email *service.MockEmailService, logger *service.MockLoggerService) {
				auth.On("CheckSession", mock.Anything).Return(testUsername, false, false, nil).Once()
				auth.On("CheckSession", mock.Anything).Return(testUsername, true, false, nil).Once()
				auth.On("CheckSession", mock.Anything).Return("", false, false, nil).Once()
				logger.On("Exception", mock.AnythingOfType("string")).Return()
				email.On("SendEmail", testEmail, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
			},
			&model.BanAPI{
				Username:   testUsername,
				Expire:     1,
				Reason:     "test",
				AdminName:  testUsername,
				Characters: nil,
			},
			http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := testRepository(t)
			defer testCleanup(t, repo)

			auth := new(service.MockAuthService)
			email := new(service.MockEmailService)
			logger := new(service.MockLoggerService)

			tt.mockFunc(auth, email, logger)

			app := testServer(service.NewUserService(repo), auth, email, service.NewCharacterService(repo), logger)

			registerAndConfirmAccount(t, app)
			createCharacter(t, app)

			resp := testSendRequest(t, app, http.MethodPost, "/restricted/ban", tt.data)
			resp = testSendRequest(t, app, http.MethodPost, "/restricted/unban", tt.data)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Unexpected status code for test: %s", tt.name)
		})
	}
}

func TestAjail(t *testing.T) {
	tests := []struct {
		name           string
		mockFunc       func(*service.MockAuthService, *service.MockEmailService, *service.MockLoggerService)
		data           *model.AjailAPI
		expectedStatus int
	}{
		{
			"User is authenticated and with correct privilege",
			func(auth *service.MockAuthService, email *service.MockEmailService, logger *service.MockLoggerService) {
				auth.On("CheckSession", mock.Anything).Return(testUsername, false, false, nil).Once()
				auth.On("CheckSession", mock.Anything).Return(testUsername, true, false, nil)
				email.On("SendEmail", testEmail, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
			},
			&model.AjailAPI{
				Character: "Test_Test",
				AdminName: "test",
				Time:      30,
				Reason:    "test",
			},
			http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := testRepository(t)
			defer testCleanup(t, repo)

			auth := new(service.MockAuthService)
			email := new(service.MockEmailService)
			logger := new(service.MockLoggerService)

			tt.mockFunc(auth, email, logger)

			app := testServer(service.NewUserService(repo), auth, email, service.NewCharacterService(repo), logger)

			registerAndConfirmAccount(t, app)
			createCharacter(t, app)

			resp := testSendRequest(t, app, http.MethodPost, "/restricted/ajail", tt.data)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Unexpected status code for test: %s", tt.name)
		})
	}
}
