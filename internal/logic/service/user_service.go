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
	repo data.IUserRepo
	rdb  *redis.Client
}

func NewUserService(repo data.IUserRepo, rdb *redis.Client) *UserService {
	return &UserService{repo: repo, rdb: rdb}
}

func (s *UserService) Register(user *model.User) (string, error) {
	logger := log.Logger

	hashedPassword, err := util.HashPassword(user.Password)
	if err != nil {
		logger.Sugar().Errorf("user_service: Password hashing error: %v", err)
		return "", err
	}

	user.Password = hashedPassword

	// 固定uuid为5位字符
	user.Uuid = uuid.New().String()[:5]
	user.CreateAt = time.Now()

	if err := s.repo.Create(user); err != nil {
		logger.Sugar().Errorf("user_service: Create user error: %v", err)
		return "", err
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
	dbUser, err := s.repo.GetUserByUsername(user.Username)
	if err != nil {
		return "", err
	}

	if res := util.ComparePassword(dbUser.Password, user.Password); res != true {
		return "", errors.New("invalid password")
	}

	// Generate JWT token
	tokenString, err := token.GenerateToken(dbUser)
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

	if user.Password != "" {
		hashed, err := util.HashPassword(user.Password)
		if err != nil {
			return err
		}
		user.Password = hashed
	}

	return s.repo.Update(user)
}

func (s *UserService) GetUserByUuid(uuid string) (*model.User, error) {
	return s.repo.GetUserByUuid(uuid)
}

func (s *UserService) GetUserByID(id uint) (*model.User, error) {
	return s.repo.GetUserByID(id)
}

func (s *UserService) GetUserStatus(userUUID string) (severID string, isOnline bool, err error) {
	severID, err = s.rdb.HGet(s.rdb.Context(), "user_status", userUUID).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", false, nil
		}
		return "", false, err
	}
	return severID, true, nil
}
