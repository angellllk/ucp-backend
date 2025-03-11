package service

import "github.com/stretchr/testify/mock"

type MockLoggerService struct {
	mock.Mock
}

func (m *MockLoggerService) Info(msg string) {
	m.Called(msg)
}

func (m *MockLoggerService) Warning(msg string) {
	m.Called(msg)
}

func (m *MockLoggerService) Exception(msg string) {
	m.Called(msg)
}

func (m *MockLoggerService) Debug(msg string) {
	m.Called(msg)
}

func (m *MockLoggerService) Shutdown() {
	return
}
