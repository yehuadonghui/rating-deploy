package models

import (
	"time"
)

// SystemConfig 系统配置（Key-Value）
type SystemConfig struct {
	Key         string    `gorm:"primaryKey;size:50" json:"key"`
	Value       string    `gorm:"type:text" json:"value"`
	Description string    `gorm:"size:255" json:"description"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TierConfig 段位配置
type TierConfig struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	Name      string `gorm:"size:50;not null" json:"name"`
	MinRating int    `gorm:"not null" json:"min_rating"`
	MaxRating int    `gorm:"not null" json:"max_rating"`
	Color     string `gorm:"size:20" json:"color"`
	Order     int    `gorm:"index" json:"order"`
}

// DefaultTiers 默认段位配置（适配从0开始的Rating）
var DefaultTiers = []TierConfig{
	{Name: "Newbie", MinRating: 0, MaxRating: 99, Color: "#808080", Order: 1},
	{Name: "Pupil", MinRating: 100, MaxRating: 199, Color: "#008000", Order: 2},
	{Name: "Specialist", MinRating: 200, MaxRating: 299, Color: "#03a89e", Order: 3},
	{Name: "Expert", MinRating: 300, MaxRating: 449, Color: "#0000ff", Order: 4},
	{Name: "Candidate Master", MinRating: 450, MaxRating: 599, Color: "#aa00aa", Order: 5},
	{Name: "Master", MinRating: 600, MaxRating: 799, Color: "#ff8c00", Order: 6},
	{Name: "Grandmaster", MinRating: 800, MaxRating: 9999, Color: "#ff0000", Order: 7},
}
