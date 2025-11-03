package data

import (
	"MyGoChat/internal/model"
	"errors"

	"gorm.io/gorm"
)

type IUserRepo interface {
	Create(user *model.User) error
	GetUserByUsername(username string) (*model.User, error)
	Update(user *model.User) error
	GetUserByUuid(uuid string) (*model.User, error)
	GetUserByID(id uint) (*model.User, error)
}

type userRepo struct {
	db *gorm.DB
}

func NewUserRepo(data *Data) IUserRepo {
	return &userRepo{db: data.Db}
}

func (r *userRepo) Create(user *model.User) error {
	var userCount int64
	if res := r.db.Model(&model.User{}).Where("username = ?", user.Username).Count(&userCount); res.Error != nil {
		return res.Error
	}
	if userCount > 0 {
		return errors.New("user already exists")
	}

	if res := r.db.Create(user); res.Error != nil {
		return res.Error
	}
	return nil
}

func (r *userRepo) GetUserByUsername(username string) (*model.User, error) {
	var dbUser model.User
	if res := r.db.Select("*").Where("username = ?", username).First(&dbUser); res.Error != nil {
		return nil, errors.New("invalid username or password")
	}
	return &dbUser, nil
}

func (r *userRepo) Update(user *model.User) error {
	updateData := make(map[string]interface{})
	if user.Nickname != "" {
		updateData["nickname"] = user.Nickname
	}
	if user.Email != "" {
		updateData["email"] = user.Email
	}
	if user.Password != "" {
		updateData["password"] = user.Password
	}

	if len(updateData) == 0 {
		return errors.New("no fields to update")
	}

	if res := r.db.Model(&model.User{}).Where("uuid = ?", user.Uuid).Updates(updateData); res.Error != nil {
		return res.Error
	}
	return nil
}

func (r *userRepo) GetUserByUuid(uuid string) (*model.User, error) {
	var user model.User
	if res := r.db.Where("uuid = ?", uuid).First(&user); res.Error != nil {
		return nil, res.Error
	}
	return &user, nil
}

func (r *userRepo) GetUserByID(id uint) (*model.User, error) {
	var user model.User
	if res := r.db.First(&user, id); res.Error != nil {
		return nil, res.Error
	}
	return &user, nil
}
