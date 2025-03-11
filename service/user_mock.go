package service

import (
	"github.com/stretchr/testify/mock"
	"sarp_backend/model"
)

type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) Create(data *model.RegisterAPI) error {
	args := m.Called(data)
	return args.Error(0)
}

func (m *MockUserService) ActivateAccount(email string) error {
	return nil
}

func (m *MockUserService) CheckActivation(name string) (bool, error) {
	return false, nil
}

func (m *MockUserService) Verify(data *model.LoginAPI) error {
	args := m.Called(data)
	return args.Error(0)
}

func (m *MockUserService) Fetch(name string) (bool, error) {
	return false, nil
}

func (m *MockUserService) IsTester(name string) (bool, error) {
	return false, nil
}

func (m *MockUserService) IsAdmin(name string) (bool, error) {
	return false, nil
}

func (m *MockUserService) GetStats(name string) (*model.GetStatsAPI, error) {
	args := m.Called(name)
	return args.Get(0).(*model.GetStatsAPI), args.Error(1)
}

func (m *MockUserService) FetchCharacter(name string) (*model.CharacterAPI, error) {
	return nil, nil
}

func (m *MockUserService) Ban(data *model.BanAPI) error {
	return nil
}

func (m *MockUserService) BanList() ([]model.BanAPI, error) {
	return nil, nil
}

func (m *MockUserService) Unban(data *model.BanAPI) error {
	return nil
}

func (m *MockUserService) DeleteExpired() error {
	return nil
}
