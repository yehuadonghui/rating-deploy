package database

import (
	"log"

	"rating-system/internal/models"
)

// Migrate 执行数据库迁移
func Migrate() error {
	log.Println("开始数据库迁移...")

	// 先确保关键表的主键类型正确（SQLite 对主键类型较敏感）
	if err := ensureBaseTables(); err != nil {
		return err
	}

	err := DB.AutoMigrate(
		&models.Student{},
		&models.Contest{},
		&models.PointRecord{},
		&models.SystemConfig{},
		&models.TierConfig{},
	)
	if err != nil {
		return err
	}

	// 初始化默认段位配置
	if err := initDefaultTiers(); err != nil {
		return err
	}

	// 初始化默认系统配置
	if err := initDefaultConfigs(); err != nil {
		return err
	}

	log.Println("数据库迁移完成")
	return nil
}

// ensureBaseTables 显式创建基础表，确保 student_id 为 TEXT
func ensureBaseTables() error {
	if !DB.Migrator().HasTable(&models.Student{}) {
		if err := DB.Exec(`
			CREATE TABLE IF NOT EXISTS students (
				student_id TEXT PRIMARY KEY,
				name TEXT NOT NULL,
				email TEXT,
				class TEXT,
				grade INTEGER,
				current_rating REAL DEFAULT 0,
				max_rating REAL DEFAULT 0,
				match_count INTEGER DEFAULT 0,
				redeem_points INTEGER DEFAULT 0,
				created_at DATETIME,
				updated_at DATETIME
			)
		`).Error; err != nil {
			return err
		}
	}

	if err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS results (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			student_id TEXT NOT NULL,
			contest_id INTEGER,
			rank INTEGER,
			solved INTEGER,
			total_time TEXT,
			performance REAL,
			rating_before REAL,
			rating_after REAL,
			delta REAL,
			created_at DATETIME
		)
	`).Error; err != nil {
		return err
	}

	if err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS point_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			student_id TEXT NOT NULL,
			contest_id INTEGER,
			category TEXT NOT NULL,
			source TEXT NOT NULL,
			points INTEGER NOT NULL,
			title TEXT NOT NULL,
			description TEXT,
			created_at DATETIME
		)
	`).Error; err != nil {
		return err
	}

	return nil
}

// initDefaultTiers 初始化默认段位
func initDefaultTiers() error {
	var count int64
	DB.Model(&models.TierConfig{}).Count(&count)
	if count > 0 {
		return nil
	}

	log.Println("初始化默认段位配置...")
	for _, tier := range models.DefaultTiers {
		if err := DB.Create(&tier).Error; err != nil {
			return err
		}
	}
	return nil
}

// initDefaultConfigs 初始化默认系统配置
func initDefaultConfigs() error {
	configs := []models.SystemConfig{
		{Key: "k_factor", Value: "48", Description: "基础变化系数"},
		{Key: "growth_inertia", Value: "0.8", Description: "变化阻尼（控制涨分速度）"},
		{Key: "history_decay", Value: "0.95", Description: "历史比赛权重衰减"},
	}

	for _, cfg := range configs {
		var existing models.SystemConfig
		result := DB.Where("key = ?", cfg.Key).First(&existing)
		if result.RowsAffected == 0 {
			if err := DB.Create(&cfg).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
