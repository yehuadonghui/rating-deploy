package models

import (
	"strconv"
	"strings"
	"time"
)

// Student 学生模型
type Student struct {
	StudentID     string    `gorm:"primaryKey;type:text;autoIncrement:false" json:"student_id"`
	Name          string    `gorm:"size:100;not null" json:"name"`
	Email         string    `gorm:"size:255" json:"email"`
	Class         string    `gorm:"size:100" json:"class"`
	Grade         int       `gorm:"index" json:"grade"`
	CurrentRating float64   `gorm:"default:0" json:"current_rating"`
	MaxRating     float64   `gorm:"default:0" json:"max_rating"`
	MatchCount    int       `gorm:"default:0" json:"match_count"`
	RedeemPoints  int       `gorm:"default:0" json:"redeem_points"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Results       []Result  `gorm:"foreignKey:StudentID" json:"results,omitempty"`
}

// ParseGrade 从学号解析年级（前2位为入学年份）
func ParseGrade(studentID string) int {
	studentID = strings.TrimSpace(studentID)
	if len(studentID) < 2 {
		return 0
	}
	grade, err := strconv.Atoi(studentID[:2])
	if err != nil {
		return 0
	}
	return grade
}

// BeforeCreate 创建前自动解析年级
func (s *Student) BeforeCreate() error {
	if s.Grade == 0 {
		s.Grade = ParseGrade(s.StudentID)
	}
	return nil
}

// UpdateRating 更新 Rating
func (s *Student) UpdateRating(newRating float64) {
	s.CurrentRating = newRating
	if newRating > s.MaxRating {
		s.MaxRating = newRating
	}
}
