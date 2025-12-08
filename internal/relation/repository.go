package relation

import (
	"MyGoChat/internal/platform"
	"MyGoChat/internal/util"
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	JoinGroupRelation(ctx context.Context, userUUID, groupUUID string) error
	CreateFriendRelation(ctx context.Context, userUUID, friendUUID string) error
	ListUserRelation(ctx context.Context, userUUID string) ([]*Relation, error)
	GetGroupMemberUUIDs(ctx context.Context, groupUUID string) ([]string, error)
}

type repository struct {
	db *gorm.DB
}

func NewRelationRepo(data *platform.Data) Repository {
	return &repository{db: data.Db}
}

func (r *repository) JoinGroupRelation(ctx context.Context, userUUID, groupUUID string) error {

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&Relation{
			UserUUID:       userUUID,
			TargetUUID:     groupUUID,
			Type:           TypeGroup,
			Status:         1, // Active
			ConversationID: groupUUID,
		}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *repository) ListUserRelation(ctx context.Context, userUUID string) ([]*Relation, error) {

	var relations []*Relation

	if err := r.db.WithContext(ctx).Where("user_uuid = ? AND status = ?", userUUID, 1).Find(&relations).Error; err != nil {
		return nil, err
	}
	return relations, nil
}

func (r *repository) CreateFriendRelation(ctx context.Context, userUUID, friendUUID string) error {
	convID := util.GetPrivateConversationID(userUUID, friendUUID)
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create relation for user
		if err := tx.Create(&Relation{
			UserUUID:       userUUID,
			TargetUUID:     friendUUID,
			Type:           TypePrivate,
			Status:         1, // Active
			ConversationID: convID,
		}).Error; err != nil {
			return err
		}

		// Create relation for friend
		if err := tx.Create(&Relation{
			UserUUID:       friendUUID,
			TargetUUID:     userUUID,
			Type:           TypePrivate,
			Status:         1, // Active
			ConversationID: convID,
		}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *repository) GetGroupMemberUUIDs(ctx context.Context, groupUUID string) ([]string, error) {
	var memberUUIDs []string
	err := r.db.WithContext(ctx).
		Model(&Relation{}).
		Where("target_uuid = ? AND type = ? AND status = ?", groupUUID, TypeGroup, 1).
		Pluck("user_uuid", &memberUUIDs).Error

	if err != nil {
		return nil, err
	}
	return memberUUIDs, nil
}
