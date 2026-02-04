package rating

import (
	"context"

	"rating-system/internal/models"
)

// Participant 参赛者信息
type Participant struct {
	StudentID     string
	CurrentRating float64
	Rank          int
	Solved        int
}

// RatingChange Rating 变化结果
type RatingChange struct {
	StudentID    string
	Performance  float64
	RatingBefore float64
	RatingAfter  float64
	Delta        float64
}

// RatingConfig Rating 算法配置
type RatingConfig struct {
	KFactor            float64 `json:"k_factor"`             // K值系数，影响单场Rating变化幅度
	GrowthInertia      float64 `json:"growth_inertia"`       // 成长阻尼系数 (0-2)
	HistoryDecay       float64 `json:"history_decay"`        // 历史衰减系数（预留）
	InitialRating      float64 `json:"initial_rating"`       // 初始Rating（所有人起点）
	TopBonusThreshold  float64 `json:"top_bonus_threshold"`  // 前X%获得额外奖励 (0-1)
	TopBonusMultiplier float64 `json:"top_bonus_multiplier"` // 前X%奖励乘数
	ParticipationBonus float64 `json:"participation_bonus"`  // 参与奖励：所有参赛者获得的基础分
	RankBonusMax       float64 `json:"rank_bonus_max"`       // 排名奖励上限：最后一名也能获得的最大分数
	RankBonusCurve     float64 `json:"rank_bonus_curve"`     // 排名奖励曲线：>1更平缓(更多人得高分)，<1更陡峭
}

// DefaultConfig 默认配置
func DefaultConfig() RatingConfig {
	return RatingConfig{
		KFactor:            48,
		GrowthInertia:      0.8,
		HistoryDecay:       0.95,
		InitialRating:      0,
		TopBonusThreshold:  0.1,
		TopBonusMultiplier: 0.5,
		ParticipationBonus: 5,   // 每场基础参与分
		RankBonusMax:       30,  // 排名奖励最大值
		RankBonusCurve:     1.5, // 曲线系数>1使分布更平缓
	}
}

// RatingCalculator Rating 计算器接口
type RatingCalculator interface {
	// Calculate 计算单场比赛后的 Rating 变化
	Calculate(ctx context.Context, contest *models.Contest, participants []Participant, config RatingConfig) ([]RatingChange, error)

	// Name 获取算法名称
	Name() string
}
