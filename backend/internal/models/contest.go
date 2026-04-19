package models

import (
	"time"
)

// Contest 比赛模型
type Contest struct {
	ID               uint              `gorm:"primaryKey" json:"id"`
	Name             string            `gorm:"size:255;not null" json:"name"`
	Date             time.Time         `gorm:"index" json:"date"`
	Weight           float64           `gorm:"default:1.0" json:"weight"`
	RewardType       ContestRewardType `gorm:"size:32;default:'none'" json:"reward_type"`
	CSVPath          string            `gorm:"size:512" json:"csv_path"`
	ParticipantCount int               `json:"participant_count"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	Results          []Result          `gorm:"foreignKey:ContestID" json:"results,omitempty"`
}
