package repository

import (
	"database/sql"
	"errors"
	"log"

	"toyStore/models"

	"golang.org/x/crypto/bcrypt"
)

type UserRepository interface {
	GetUserById(id int) (models.User_db, bool, error)
	GetUserByName(name string) (models.User_db, bool, error)
	EncryptPassword(userPass string) (hashedPassword string, err error)
	VerifyPassword(hashedPassword string, sentPassword string) bool
	UpdatePassword(userId int, newPassword string) error
	AddNewUser(uModel models.User_db) (newUserId int, err error)
}

type UserRepo struct {
	db *sql.DB
}

func NewUserRepository(conn *sql.DB) (UserRepository, error) {
	if conn == nil {
		return nil, errors.New("conn must be non-nil")
	}
	err := conn.Ping()
	if err != nil {
		return nil, err
	}
	return &UserRepo{
		db: conn,
	}, nil
}

func (u *UserRepo) GetUserById(id int) (uModel models.User_db, exists bool, err error) {
	row := u.db.QueryRow("select Id, Nickname, Password, Role from Users where Id = $1", id)
	err = row.Scan(&uModel.Id, &uModel.Nickname, &uModel.Password, &uModel.Role)
	if err != nil {
		if err == sql.ErrNoRows {
			err = nil
			return
		}
		log.Printf("GetUserById: %v", err)
		err = models.ErrServerError
	}
	exists = true
	return
}

func (u *UserRepo) GetUserByName(name string) (uModel models.User_db, exists bool, err error) {
	row := u.db.QueryRow("select Id, Nickname, Password, Role from Users where Nickname = $1", name)
	err = row.Scan(&uModel.Id, &uModel.Nickname, &uModel.Password, &uModel.Role)
	if err != nil {
		if err == sql.ErrNoRows {
			err = nil
			return
		}
		log.Printf("GetUserByName: %v", err)
		err = models.ErrServerError
	}
	exists = true
	return
}

func (u *UserRepo) EncryptPassword(userPass string) (hashedPassword string, err error) {
	var password []byte
	password, err = bcrypt.GenerateFromPassword([]byte(userPass), 8)
	if err != nil {
		log.Printf("EncryptPassword: %v", err)
		err = models.ErrServerError
		return
	}
	hashedPassword = string(password)
	return
}

func (u *UserRepo) VerifyPassword(hashedPassword string, sentPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(sentPassword))
	if err != nil {
		log.Printf("VerifyPassword: %v", err)
	}
	return err == nil
}

func (u *UserRepo) AddNewUser(uModel models.User_db) (newUserId int, err error) {
	err = u.db.QueryRow("INSERT INTO Users (Nickname, Password, Role) VALUES ($1, $2, $3) RETURNING id;", uModel.Nickname, uModel.Password, uModel.Role).Scan(newUserId)
	if err != nil {
		log.Printf("AddNewUser: %v", err)
		err = models.ErrServerError
	}
	return
}

func (u *UserRepo) UpdatePassword(userId int, newPassword string) error {
	_, err := u.db.Exec("UPDATE Users SET Password = $1 WHERE Id = $2;", newPassword, userId)
	if err != nil {
		log.Printf("UpdatePassword: %v", err)
		err = models.ErrServerError
	}
	return err
}
