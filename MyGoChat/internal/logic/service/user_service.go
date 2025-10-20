package service

import (
	"MyGoChat/internal/model"
	"MyGoChat/pkg/db"
	"MyGoChat/pkg/log"
	"errors"
	"time"

	"github.com/google/uuid"
)

type userService struct {
}

var UserService = new(userService)

func (u *userService) Register(user *model.User) error {
	logger := log.Logger
	d := db.GetDB()

	// 确保表存在
	if err := d.AutoMigrate(&model.User{}); err != nil {
		logger.Sugar().Errorf("user_service: AutoMigrate error: %v", err)
		return err
	}

	var userCount int64
	if res := d.Model(&model.User{}).Where("username = ?", user.Username).Count(&userCount); res.Error != nil {
		return res.Error
	}
	if userCount > 0 {
		return errors.New("user already exists")
	}

	user.Uuid = uuid.New().String()
	user.CreateAt = time.Now()
	user.DeleteAt = 0

	if res := d.Create(user); res.Error != nil {
		logger.Sugar().Errorf("user_service: Create user error: %v", res.Error)
		return res.Error
	}
	logger.Sugar().Infof("user_service: User registered successfully: %v", user.Username)
	return nil
}
