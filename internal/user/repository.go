package user

import (
	"MyGoChat/internal/platform"
	"context"
	"errors"

	"gorm.io/gorm"
)

type Repository interface {
	Create(user *User) error
	GetUserByUsername(username string) (*User, error)
	Update(user *User) error
	GetUserByUuid(uuid string) (*User, error)
	GetUserByID(id uint) (*User, error)
	GetUUIDByUsername(ctx context.Context, username string) (string, error)
}

type repository struct {
	db *gorm.DB
}

func NewUserRepo(data *platform.Data) Repository {
	return &repository{db: data.Db}
}

func (r *repository) Create(user *User) error {
	var userCount int64
	if res := r.db.Model(&User{}).Where("username = ?", user.Username).Count(&userCount); res.Error != nil {
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

func (r *repository) GetUserByUsername(username string) (*User, error) {
	var dbUser User
	if res := r.db.Select("*").Where("username = ?", username).First(&dbUser); res.Error != nil {
		return nil, errors.New("invalid username or password")
	}
	return &dbUser, nil
}

func (r *repository) Update(user *User) error {
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

	if res := r.db.Model(&User{}).Where("uuid = ?", user.Uuid).Updates(updateData); res.Error != nil {
		return res.Error
	}
	return nil
}

func (r *repository) GetUserByUuid(uuid string) (*User, error) {
	var user User
	if res := r.db.Where("uuid = ?", uuid).First(&user); res.Error != nil {
		return nil, res.Error
	}
	return &user, nil
}

func (r *repository) GetUserByID(id uint) (*User, error) {
	var user User
	if res := r.db.First(&user, id); res.Error != nil {
		return nil, res.Error
	}
	return &user, nil
}

func (r *repository) GetUUIDByUsername(ctx context.Context, username string) (string, error) {
	var user User
	// 只查 uuid 字段，性能更高
	err := r.db.WithContext(ctx).
		Select("uuid").
		Where("username = ?", username).
		First(&user).Error

	if err != nil {
		return "", err
	}
	return user.Uuid, nil
}
