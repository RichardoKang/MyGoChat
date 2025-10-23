package service

import (
	"MyGoChat/internal/model"
	"MyGoChat/internal/util"
	"MyGoChat/pkg/db"
	"MyGoChat/pkg/log"
	"MyGoChat/pkg/token"
	"errors"
	"time"

	"github.com/google/uuid"
)

type userService struct {
}

var UserService = new(userService)

func (u *userService) Register(user *model.User) (string, error) {
	logger := log.Logger
	d := db.GetDB()

	// 确保表存在
	if err := d.AutoMigrate(&model.User{}); err != nil {
		logger.Sugar().Errorf("user_service: AutoMigrate error: %v", err)
		return "", err
	}

	var userCount int64
	if res := d.Model(&model.User{}).Where("username = ?", user.Username).Count(&userCount); res.Error != nil {
		return "", res.Error
	}
	if userCount > 0 {
		return "", errors.New("user already exists")
	}

	hashedPassword, err := util.HashPassword(user.Password)
	if err != nil {
		logger.Sugar().Errorf("user_service: Password hashing error: %v", err)
		return "", err
	}

	user.Password = hashedPassword
	user.Uuid = uuid.New().String()
	user.CreateAt = time.Now()

	if res := d.Create(user); res.Error != nil {
		logger.Sugar().Errorf("user_service: Create user error: %v", res.Error)
		return "", res.Error
	}
	logger.Sugar().Infof("user_service: User registered successfully: %v", user.Username)

	// Generate JWT token
	tokenString, err := token.GenerateToken(user)
	if err != nil {
		logger.Sugar().Errorf("user_service: Failed to generate token for user %s: %v", user.Username, err)
		return "", err
	}

	logger.Sugar().Infof("user_service: User %s registered and token generated", user.Username)

	return tokenString, nil
}

func (u *userService) Login(user *model.User) (string, error) {
	d := db.GetDB()
	var dbUser model.User
	if res := d.Select("*").Where("username = ?", user.Username).First(&dbUser); res.Error != nil {
		return "", errors.New("invalid username or password")
	}

	// Explicitly fetch the password as it might be ignored due to json:"-" tag
	var password string
	if res := d.Model(&model.User{}).Select("password").Where("username = ?", user.Username).Scan(&password); res.Error != nil {
		return "", errors.New("invalid username or password") // Or a more specific error if needed
	}
	dbUser.Password = password // Assign the fetched password

	if res := util.ComparePassword(dbUser.Password, user.Password); res != true {
		return "", errors.New("invalid password")
	}

	// Generate JWT token
	tokenString, err := token.GenerateToken(&dbUser)
	if err != nil {
		log.Logger.Sugar().Errorf("user_service: Failed to generate token for user %s: %v", user.Username, err)
		return "", err
	}

	return tokenString, nil
}

func (u *userService) Update(user *model.User, userUuid string) error {
	user.Uuid = userUuid

	if user.Uuid == "" {
		return errors.New("invalid user data")
	}

	d := db.GetDB()

	updateData := map[string]any{}
	if user.Nickname != "" {
		updateData["nickname"] = user.Nickname
	}
	if user.Email != "" {
		updateData["email"] = user.Email
	}
	if user.Password != "" {
		hashed, err := util.HashPassword(user.Password)
		if err != nil {
			return err
		}
		updateData["password"] = hashed
	}

	if len(updateData) == 0 {
		return errors.New("no fields to update")
	}

	if res := d.Model(&model.User{}).Where("uuid = ?", user.Uuid).Updates(updateData); res.Error != nil {
		return res.Error
	}
	return nil
}
