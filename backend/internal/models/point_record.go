package models

import "time"

type ContestRewardType string

const (
	ContestRewardTypeNone           ContestRewardType = "none"
	ContestRewardTypeFoundationExam ContestRewardType = "foundation_exam"
	ContestRewardTypeBrandMonthly   ContestRewardType = "brand_monthly"
	ContestRewardTypeBrandFinal     ContestRewardType = "brand_final"
)

type PointCategory string

const (
	PointCategoryContestAward    PointCategory = "contest_award"
	PointCategoryProgressAward   PointCategory = "progress_award"
	PointCategoryOrganizerReward PointCategory = "organizer_reward"
)

type PointSource string

const (
	PointSourceAuto   PointSource = "auto"
	PointSourceManual PointSource = "manual"
)

type PointRecord struct {
	ID          uint          `gorm:"primaryKey" json:"id"`
	StudentID   string        `gorm:"index;type:text;not null" json:"student_id"`
	ContestID   *uint         `gorm:"index" json:"contest_id,omitempty"`
	Category    PointCategory `gorm:"size:32;not null" json:"category"`
	Source      PointSource   `gorm:"size:16;not null" json:"source"`
	Points      int           `gorm:"not null" json:"points"`
	Title       string        `gorm:"size:255;not null" json:"title"`
	Description string        `gorm:"size:1024" json:"description"`
	CreatedAt   time.Time     `json:"created_at"`
	Student     Student       `gorm:"foreignKey:StudentID" json:"student,omitempty"`
	Contest     Contest       `gorm:"foreignKey:ContestID" json:"contest,omitempty"`
}
