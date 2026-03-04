package platform

import (
	"MyGoChat/pkg/log"

	"gorm.io/gorm"
)

// AutoMigrate 自动迁移数据库表结构
// 注意：模型定义需要在调用此函数时传入，避免循环导入
func AutoMigrate(db *gorm.DB, models ...interface{}) error {
	if err := db.AutoMigrate(models...); err != nil {
		log.Logger.Sugar().Errorf("数据库自动迁移失败: %v", err)
		return err
	}
	log.Logger.Sugar().Info("数据库自动迁移成功")
	return nil
}
