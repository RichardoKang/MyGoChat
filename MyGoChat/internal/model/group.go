package model

import (
	"time"
)

type Group struct {
	ID          uint      `json:"id" gorm:"primarykey"`
	Uuid        string    `json:"uuid" gorm:"index;unique;not null"`
	Name        string    `json:"name" gorm:"type:varchar(150);not null;comment:'群名称'"`
	AdminUserID uint      `json:"adminUserId" gorm:"not null;comment:'群主ID'"`
	AdminUser   User      `json:"adminUser" gorm:"foreignKey:AdminUserID"`
	CreatedAt   time.Time `json:"createdAt"`
}
