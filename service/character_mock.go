package service

import (
	"github.com/stretchr/testify/mock"
	"sarp_backend/model"
)

type MockCharacterService struct {
	mock.Mock
}

func (c *MockCharacterService) Create(data *model.CharacterDataAPI) error {
	return nil
}

func (c *MockCharacterService) FetchWaiting() ([]model.CharacterDataAPI, error) {
	return nil, nil
}

func (c *MockCharacterService) AcceptCharacter(data model.CharacterAPI) error {
	return nil
}

func (c *MockCharacterService) DeclineCharacter(data model.RejectCharacterAPI) error {
	return nil
}
