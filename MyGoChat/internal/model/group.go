package model

import "time"

type Group struct {
	GroupId     int32     `json:"groupId" gorm:"primary_key;AUTO_INCREMENT;comment:'id'"`
	Uuid        string    `json:"uuid" gorm:"type:varchar(150);not null;unique_index:idx_uuid;comment:'uuid'"`
	Name        string    `json:"name" gorm:"type:varchar(150);not null;comment:'群名称'"`
	AdminUserId int32     `json:"adminUserId" gorm:"not null;comment:'群主ID'"`
	AdminUser   User      `json:"adminUser" gorm:"foreignKey:AdminUserID;references:ID;comment:'群主'"`
	CreateAt    time.Time `json:"createAt"`
}
