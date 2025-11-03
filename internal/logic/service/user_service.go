package service

import (
	"MyGoChat/internal/logic/data"
	"MyGoChat/internal/model"
	"MyGoChat/internal/util"
	"MyGoChat/pkg/log"
	"MyGoChat/pkg/token"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

type UserService struct {
	data *data.Data
}

func NewUserService(data *data.Data) *UserService {
	return &UserService{data: data}
}

func (s *UserService) Register(user *model.User) (string, error) {
	logger := log.Logger
	d := s.data.GetDB() // 使用注入的 data

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

func (s *UserService) Login(user *model.User) (string, error) {
	d := s.data.GetDB() // 使用注入的 data
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

func (s *UserService) Update(user *model.User, userUuid string) error {
	user.Uuid = userUuid

	if user.Uuid == "" {
		return errors.New("invalid user data")
	}

	d := s.data.GetDB() // 使用注入的 data

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

func (s *UserService) GetUserByUuid(uuid string) (*model.User, error) {
	d := s.data.GetDB() // 使用注入的 data
	var user model.User
	if res := d.Where("uuid = ?", uuid).First(&user); res.Error != nil {
		return nil, res.Error
	}
	return &user, nil
}

func (s *UserService) GetUserByID(id uint) (*model.User, error) {
	d := s.data.GetDB() // 使用注入的 data
	var user model.User
	if res := d.First(&user, id); res.Error != nil {
		return nil, res.Error
	}
	return &user, nil
}

func (s *UserService) GetUserStatus(userID uint) (severID string, isOnline bool, err error) {
	rdb := s.data.GetRedisClient()
	severID, err = rdb.HGet(rdb.Context(), "user_status", string(rune(userID))).Result()
	if err == nil {
		if errors.Is(err, redis.Nil) {
			return "", false, err
		}
		return "", false, err
	}
	return severID, true, nil
}
