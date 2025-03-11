package service

import (
	"encoding/hex"
	"errors"
	"github.com/jzelinskie/whirlpool"
	"sarp_backend/model"
	"sarp_backend/repository"
	"strings"
	"time"
)

type UserService struct {
	userRepository *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{userRepository: repo}
}

func (u *UserService) Create(data *model.RegisterAPI) error {
	dto := &repository.UserDB{
		Username:     data.Username,
		Email:        data.Email,
		Password:     hashWP(data.Password),
		RegisterDate: time.Now().Format("2006-01-02 15:04:05"),
	}

	return u.userRepository.Create(dto)
}

func (u *UserService) ActivateAccount(email string) error {
	return u.userRepository.Activate(email)
}

func (u *UserService) CheckActivation(name string) (bool, error) {
	return u.userRepository.CheckActivation(name)
}

func (u *UserService) Verify(data *model.LoginAPI) error {
	dto := &repository.UserDB{
		Username: data.Username,
		Password: hashWP(data.Password),
	}

	return u.userRepository.Verify(dto)
}

func (u *UserService) UpdatePassword(email string, password string) error {
	hashed := hashWP(password)
	return u.userRepository.UpdatePassword(email, hashed)
}

func (u *UserService) Fetch(name string, email string) (bool, error) {
	return u.userRepository.Fetch(name, email)
}

func (u *UserService) FetchMail(name string) (string, error) {
	return u.userRepository.FetchMail(name)
}

func (u *UserService) IsTester(name string) (bool, error) {
	return u.userRepository.FetchTesterLevel(name)
}

func (u *UserService) IsAdmin(name string) (bool, error) {
	return u.userRepository.FetchAdminLevel(name)
}

func (u *UserService) GetStats(name string) (*model.GetStatsAPI, error) {
	data, err := u.userRepository.FetchStats(name)
	if err != nil {
		return nil, err
	}

	if data == nil {
		return nil, nil
	}

	var list []model.CharacterStatsAPI
	for _, c := range data.CharactersData {
		dto := model.CharacterStatsAPI{
			Name:         c.Name,
			Created:      c.Created,
			Level:        c.Level,
			PlayingHours: c.PlayingHours,
		}
		list = append(list, dto)
	}

	return &model.GetStatsAPI{
		Username:      data.Username,
		Admin:         data.Admin,
		Tester:        data.Tester,
		DonateRank:    data.DonateRank,
		Characters:    data.Characters,
		LastLogin:     data.LoginDate,
		CharacterList: list,
	}, nil
}

func (u *UserService) GetStaff() ([]model.GetStaffAPI, error) {
	staff, err := u.userRepository.FetchStaff()
	if err != nil {
		return nil, err
	}

	var list []model.GetStaffAPI
	for _, c := range staff {
		dto := model.GetStaffAPI{
			Username: c.Username,
			Role:     "Tester",
		}
		if c.Admin != 0 {
			dto.Role = "Admin"
		}
		list = append(list, dto)
	}

	return list, nil
}

func (u *UserService) GetServerStats() (*model.ServerStatsAPI, error) {
	data, err := u.userRepository.FetchServerStats()
	if err != nil {
		return nil, err
	}

	return &model.ServerStatsAPI{
		Online:     data.Online,
		Bans:       data.Bans,
		Houses:     data.Houses,
		Staff:      data.Staff,
		Accounts:   data.Accounts,
		Characters: data.Characters,
	}, nil
}

func (u *UserService) FetchCharacter(name string) (*model.CharacterAPI, error) {
	if name == "" {
		return nil, errors.New("character name can't be empty")
	}

	fetch, err := u.userRepository.FetchCharacter(name)
	if err != nil {
		return nil, err
	}

	return &model.CharacterAPI{
		Username:      fetch.Username,
		CharacterName: fetch.Character,
	}, nil
}

func (u *UserService) CheckForBan(name string) (bool, error) {
	return u.userRepository.CheckForBan(name)
}

func (u *UserService) Ban(data *model.BanAPI) error {
	if data.AdminName == "" || data.Username == "" || data.Reason == "" {
		return errors.New("fields can't be empty")
	}

	if data.Expire <= 0 || data.Expire >= 30 {
		return errors.New("invalid expire value")
	}

	ban := &repository.BlacklistDB{
		Username: data.Username,
		BannedBy: data.AdminName,
		Reason:   data.Reason,
		Date:     time.Now().Format("2006-01-02 15:04:05"),
		Expire:   data.Expire,
	}

	return u.userRepository.AddBan(ban)
}

func (u *UserService) BanList() ([]model.BanAPI, error) {
	bans, err := u.userRepository.FetchBans()
	if err != nil {
		return nil, err
	}

	var ret []model.BanAPI
	for _, ban := range bans {
		var allChars []model.CharacterAPI
		for _, char := range ban.Characters {
			allChars = append(allChars, model.CharacterAPI{CharacterName: char.Character})
		}

		ret = append(ret, model.BanAPI{
			Username:   ban.Username,
			Expire:     ban.Expire,
			Reason:     ban.Reason,
			AdminName:  ban.BannedBy,
			Characters: allChars,
		})
	}

	return ret, nil
}

func (u *UserService) Unban(data *model.BanAPI) error {
	return u.userRepository.Unban(data.Username)
}

func (u *UserService) Ajail(data *model.AjailAPI) error {
	ajail := &repository.AjailDB{
		Character: data.Character,
		Prisoned:  0,
		JailTime:  data.Time * 60, // minutes
	}

	return u.userRepository.Ajail(ajail)
}

func (u *UserService) Logs(data *model.LogsAPI) ([]map[string]interface{}, error) {
	var logsType = []string{
		"logs_ajail", "logs_ban", "logs_warn", "logs_kick", "logs_unban",
		"logs_charity", "hit_logs", "logs_ck", "logs_transfer",
		"logs_givecash", "logs_givedrug", "logs_givegun", "logs_pay",
		"logs_ask", "logs_report", "namechanges"}
	var ok bool
	for _, l := range logsType {
		if strings.EqualFold(data.Type, l) {
			ok = true
			break
		}
	}
	if !ok {
		return nil, errors.New("invalid log type")
	}

	return u.userRepository.FetchLogs(data.Type)
}

func (u *UserService) DeleteExpired() error {
	return u.userRepository.DeleteExp()
}

func hashWP(payload string) string {
	w := whirlpool.New()
	w.Write([]byte(payload))
	return hex.EncodeToString(w.Sum(nil))
}
