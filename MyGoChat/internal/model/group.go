package model

import "time"

type Group struct {
	ID            int32     `json:"groupId" gorm:"primary_key;AUTO_INCREMENT;comment:'id'"`
	Uuid          string    `json:"uuid" gorm:"type:varchar(150);not null;unique_index:idx_uuid;comment:'uuid'"`
	GroupName     string    `json:"name" gorm:"type:varchar(150);not null;comment:'群名称'"`
	AdminUserUuid string    `json:"adminUserUuid" gorm:"not null;comment:'群主ID'"`
	AdminUser     User      `json:"adminUser" gorm:"foreignKey:AdminUserID;references:ID;comment:'群主'"`
	CreateAt      time.Time `json:"createAt"`
}
