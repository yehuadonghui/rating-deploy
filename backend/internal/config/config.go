package config

import (
	"os"
	"strconv"
)

// Config 应用配置
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Admin    AdminConfig
	Rating   RatingConfig
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port string
	Mode string // debug, release, test
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Path string
}

// AdminConfig 管理员配置
type AdminConfig struct {
	Password  string
	JWTSecret string
	TokenExp  int // 小时
}

// RatingConfig Rating 算法配置
type RatingConfig struct {
	KFactor       float64 `json:"k_factor"`
	GrowthInertia float64 `json:"growth_inertia"`
	HistoryDecay  float64 `json:"history_decay"`
}

// DefaultRatingConfig 默认 Rating 配置
func DefaultRatingConfig() RatingConfig {
	return RatingConfig{
		KFactor:       48,
		GrowthInertia: 0.8,
		HistoryDecay:  0.95,
	}
}

// Load 加载配置
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
			Mode: getEnv("GIN_MODE", "debug"),
		},
		Database: DatabaseConfig{
			Path: getEnv("DB_PATH", "../data/rating.db"),
		},
	Admin: AdminConfig{
			Password:  getEnv("ADMIN_PASSWORD", "$2a$10$SIgiHR4hBfKZui0YK5dw8uXONvfor4MJUGS.4wB1BHkm4UirOPKKu"),
			JWTSecret: getEnv("JWT_SECRET", "rating-system-secret-key"),
			TokenExp:  getEnvInt("TOKEN_EXP", 24),
		},
		Rating: DefaultRatingConfig(),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}
