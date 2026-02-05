package models

import (
	"time"
)

// Result 比赛结果模型
type Result struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	StudentID    string    `gorm:"index;type:text;not null" json:"student_id"`
	ContestID    uint      `gorm:"index" json:"contest_id"`
	Rank         int       `json:"rank"`
	Solved       int       `json:"solved"`
	TotalTime    string    `json:"total_time"`
	Performance  float64   `json:"performance"`
	RatingBefore float64   `json:"rating_before"`
	RatingAfter  float64   `json:"rating_after"`
	Delta        float64   `json:"delta"`
	CreatedAt    time.Time `json:"created_at"`
	Student      Student   `gorm:"foreignKey:StudentID" json:"student,omitempty"`
	Contest      Contest   `gorm:"foreignKey:ContestID" json:"contest,omitempty"`
}
