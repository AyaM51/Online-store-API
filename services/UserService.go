package services

import (
	"log"
	"time"
	"toyStore/models"
	"toyStore/repository"
)

type UserService struct {
	ur repository.UserRepository
	sr repository.SessionRepository
}

func NewUserService(uRepo repository.UserRepository, sRepo repository.SessionRepository) UserService {
	return UserService{
		ur: uRepo,
		sr: sRepo,
	}
}

func (us *UserService) SignupRequest(creds models.Credentials) (uModel models.User_db, err error) {
	var hashedPassword string
	uModel.Nickname = creds.Username
	uModel.Password = creds.Password
	if creds.Role == "" {
		creds.Role = "user"
	}
	uModel.Role = creds.Role

	var ex bool
	_, ex, err = us.ur.GetUserByName(uModel.Nickname)
	if err != nil {
		return
	}
	if ex {
		log.Printf("SignupRequest: user already exists")
		err = models.ErrNotAllowed
		return
	}
	hashedPassword, err = us.ur.EncryptPassword(uModel.Password)
	if err != nil {
		return
	}
	uModel.Password = hashedPassword
	uModel.Id, err = us.ur.AddNewUser(uModel)
	if err != nil {
		return
	}
	return
}

func (us *UserService) SigninRequest(name, password string) (uModel models.User_db, sessionId string, err error) {
	uModel.Nickname = name
	uModel.Password = password
	var ex bool
	uModel, ex, err = us.ur.GetUserByName(uModel.Nickname)
	if err != nil {
		return
	}
	if !ex {
		log.Printf("user not found")
		err = models.ErrNotAllowed
		return
	}
	if !us.ur.VerifyPassword(uModel.Password, password) {
		log.Printf("wrong password")
		err = models.ErrUnautorized
		return
	}
	sessionId, err = us.sr.CreateSession(uModel.Id, uModel.Role)
	return
}

func (us *UserService) RefreshRequest(sessionId string) (err error) {
	err = us.sr.RefreshSession(sessionId, 30*time.Minute)
	return
}

func (us *UserService) CreateUserRequest(creds models.Credentials) (err error) {
	_, err = us.SignupRequest(creds)
	return
}

func (us *UserService) CheckAccess(sessionId string) (access bool, err error) {
	_, role, exists, e := us.sr.GetUserSessionInfo(sessionId)
	if e != nil {
		err = e
		return
	}
	if !exists || role != "manager" {
		return
	}
	access = true
	return
}

func (us *UserService) CheckAuth(sessionId string) (bool, error) {
	autorized, err := us.sr.CheckSession(sessionId)
	return autorized, err
}

func (us *UserService) DeleteSessionRequest(sessionId string) (err error) {
	err = us.sr.DeleteSession(sessionId)
	return
}

func (us *UserService) WelcomeRequest(sessionId string) (uModel models.User_db, ex bool) {
	userId, _, exist, err := us.sr.GetUserSessionInfo(sessionId)
	if err != nil || !exist {
		return
	}
	uModel, exist, err = us.ur.GetUserById(userId)
	if err != nil || !exist {
		return
	}
	ex = true
	return
}

func (us *UserService) ChangePasswordRequest(sessionId string, oldPass string, newPass string) (err error) {
	var uModel models.User_db
	userId, _, _, e := us.sr.GetUserSessionInfo(sessionId)
	if e != nil {
		err = e
		return
	}

	uModel, _, err = us.ur.GetUserById(userId)
	if err != nil {
		return
	}

	if !us.ur.VerifyPassword(uModel.Password, oldPass) {
		err = models.ErrBadRequest
		return
	}
	newPass, err = us.ur.EncryptPassword(newPass)
	if err != nil {
		return
	}
	err = us.ur.UpdatePassword(userId, newPass)
	if err != nil {
		return
	}

	err = us.sr.DeleteSession(sessionId)
	return
}
