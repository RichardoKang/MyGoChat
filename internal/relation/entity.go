package relation

import (
	"time"

	"gorm.io/plugin/soft_delete"
)

const (
	TypePrivate = 1 // 私聊（好友）
	TypeGroup   = 2 // 群聊（群成员）
)

// Relation 是核心表，代表“我与某个人/群”的关系
// 索引建议：(OwnerUUID, Type), (OwnerUUID, TargetUUID)
type Relation struct {
	ID       uint   `gorm:"primarykey"`
	UserUUID string `gorm:"type:varchar(32);index;not null;comment:'谁的关系'"`

	// 目标ID：如果是私聊，这里是对方的UUID；如果是群聊，这里是群的UUID
	TargetUUID string `gorm:"type:varchar(32);index;not null;comment:'对方ID/群ID'"`

	// 关系类型：1=私聊(好友), 2=群聊(群成员)
	Type int `gorm:"type:smallint;not null;comment:'1=人, 2=群'"`

	// --- 关键优化：直接存储会话ID ---
	// 这样首页列表只需要查这一张表，就能拿到 ConversationID 去 Mongo 查未读数
	ConversationID string `gorm:"type:varchar(64);not null;comment:'关联的会话ID'"`

	// --- 通用属性 ---
	Remark string `gorm:"type:varchar(64);comment:'好友备注/群内昵称'"`

	// --- 角色与状态 (字段复用) ---
	// 私聊：0=申请中, 1=正常, 2=拉黑
	// 群聊：0=普通成员, 1=管理员, 2=群主
	Status int `gorm:"default:1;comment:'状态或角色'"`

	// --- 个人设置 (偏好) ---
	IsTop     bool                  `gorm:"default:false;comment:'置顶'"`
	IsMute    bool                  `gorm:"default:false;comment:'免打扰'"`
	DeletedAt soft_delete.DeletedAt `gorm:"index"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
