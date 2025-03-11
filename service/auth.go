package service

import (
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"time"
)

type AuthService struct {
	Store *session.Store
}

func NewAuthService(store *session.Store) *AuthService {
	return &AuthService{Store: store}
}

func (a *AuthService) CheckSession(ctx *fiber.Ctx) (string, bool, bool, error) {
	var name string
	var isAdmin, isTester bool

	sess, err := a.Store.Get(ctx)
	if err != nil {
		globalLogger.Exception(err.Error())
		return name, isAdmin, isTester, err
	}

	if r := sess.Get("name"); r != nil {
		var ok bool
		name, ok = r.(string)
		if !ok {
			errMsg := fmt.Sprintf("can't type cast to string session for user %s", name)
			globalLogger.Exception(errMsg)
			return name, isAdmin, isTester, errors.New(errMsg)
		}
	}

	if r := sess.Get("is_admin"); r != nil {
		var ok bool
		isAdmin, ok = r.(bool)
		if !ok {
			errMsg := fmt.Sprintf("can't type cast to bool is_admin for user %s", name)
			globalLogger.Exception(errMsg)
			return name, isAdmin, isTester, errors.New(errMsg)
		}
	}

	if r := sess.Get("is_tester"); r != nil {
		var ok bool
		isTester, ok = r.(bool)
		if !ok {
			errMsg := fmt.Sprintf("can't type cast to bool is_tester for user %s", name)
			globalLogger.Exception(errMsg)
			return name, isAdmin, isTester, errors.New(errMsg)
		}
	}

	return name, isAdmin, isTester, nil
}

func (a *AuthService) SaveSession(ctx *fiber.Ctx, name string, isTester bool, isAdmin bool) error {
	sess, err := a.Store.Get(ctx)
	if err != nil {
		globalLogger.Exception(err.Error())
		return err
	}
	sess.Set("name", name)
	sess.Set("is_tester", isTester)
	sess.Set("is_admin", isAdmin)
	sess.SetExpiry(time.Hour * 24)
	return sess.Save()
}

func (a *AuthService) DestroySession(ctx *fiber.Ctx) error {
	sess, err := a.Store.Get(ctx)
	if err != nil {
		globalLogger.Exception(err.Error())
		return err
	}

	return sess.Destroy()
}

func (a *AuthService) Authenticate(ctx *fiber.Ctx) error {
	name, isAdmin, isTester, err := a.CheckSession(ctx)
	if err != nil {
		return err
	}

	if name == "" {
		return errors.New("WaitingList(): session doesn't exist: user is not logged in")
	}

	if !isAdmin || !isTester {
		return errors.New(fmt.Sprintf("WaitingList(): user %s doesn't have admin rights", name))
	}

	return nil
}
