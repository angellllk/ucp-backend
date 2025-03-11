package service

import (
	"errors"
	"sarp_backend/model"
	"sarp_backend/repository"
)

type CharacterService struct {
	userRepository *repository.UserRepository
}

func NewCharacterService(repo *repository.UserRepository) *CharacterService {
	return &CharacterService{userRepository: repo}
}

func (c *CharacterService) Create(data *model.CharacterDataAPI) error {
	dto := &repository.CharacterDB{
		Username:  data.Username,
		Character: data.CharacterName,
		Age:       data.CharacterAge,
		Gender:    data.CharacterGender,
		Origin:    data.CharacterOrigin,
	}

	if dto.Gender == 1 {
		dto.Skin = 98
	} else {
		dto.Skin = 93
	}

	return c.userRepository.CreateCharacter(dto)
}

func (c *CharacterService) FetchWaiting() ([]model.CharacterDataAPI, error) {
	dataList, err := c.userRepository.FetchWaitingCharacters()
	if err != nil {
		return nil, err
	}

	var dto []model.CharacterDataAPI

	for _, data := range dataList {
		dto = append(dto, model.CharacterDataAPI{
			Username:        data.Username,
			CharacterName:   data.Character,
			CharacterAge:    data.Age,
			CharacterGender: data.Gender,
			CharacterOrigin: data.Origin,
		})
	}

	return dto, nil
}

func (c *CharacterService) AcceptCharacter(data model.CharacterAPI) error {
	if data.CharacterName == "" {
		return errors.New("unexpected character name data")
	}
	return c.userRepository.AcceptCharacter(data.Username, data.CharacterName, data.AcceptedBy)
}

func (c *CharacterService) DeclineCharacter(data model.RejectCharacterAPI) error {
	if data.CharacterName == "" {
		return errors.New("unexpected character name data")
	}
	return c.userRepository.DeclineCharacter(data.CharacterName)
}
