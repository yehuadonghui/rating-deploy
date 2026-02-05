package database

import (
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"rating-system/internal/config"
)

var DB *gorm.DB

// Init 初始化数据库连接
func Init(cfg *config.Config) error {
	var err error
	DB, err = gorm.Open(sqlite.Open(cfg.Database.Path), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Info),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return err
	}
	log.Printf("数据库连接成功: %s", cfg.Database.Path)
	return nil
}

// GetDB 获取数据库实例
func GetDB() *gorm.DB {
	return DB
}
