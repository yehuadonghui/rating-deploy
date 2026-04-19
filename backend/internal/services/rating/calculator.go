package rating

import (
	"context"
	"math"
	"sort"

	"rating-system/internal/models"
)

// EloMMRCalculator Modified Elo-MMR 变体算法
type EloMMRCalculator struct{}

func NewEloMMRCalculator() *EloMMRCalculator {
	return &EloMMRCalculator{}
}

func (c *EloMMRCalculator) Name() string {
	return "Modified Elo-MMR"
}

// Calculate 计算 Rating 变化
// Growth-Focused Rating (GFR) 算法：
// 1. 参与奖励：所有参赛者获得基础分
// 2. 排名梯度奖励：使用可调曲线，让更多人获得正向激励
// 3. 技能差异奖励：超出期望排名时额外加分
// 4. Top奖励：前X%选手额外加成
// 5. 允许下降：delta 可为负数
func (c *EloMMRCalculator) Calculate(
	ctx context.Context,
	contest *models.Contest,
	participants []Participant,
	config RatingConfig,
) ([]RatingChange, error) {
	if len(participants) == 0 {
		return nil, nil
	}

	n := float64(len(participants))

	// 计算所有参赛者的平均 Rating
	var sumRating float64
	for _, p := range participants {
		sumRating += p.CurrentRating
	}
	avgRating := sumRating / n

	// 基础分使用参赛者平均Rating，但至少为InitialRating
	baseRating := avgRating
	if baseRating < config.InitialRating {
		baseRating = config.InitialRating
	}

	// K 值乘以比赛权重
	k := config.KFactor * contest.Weight

	// 计算期望排名（按当前 Rating 排序）
	expectedRankMap := make(map[string]float64)
	sorted := make([]Participant, len(participants))
	copy(sorted, participants)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].CurrentRating > sorted[j].CurrentRating
	})
	for i, p := range sorted {
		expectedRankMap[p.StudentID] = float64(i + 1)
	}

	// 处理并列名次：相同名次的选手使用“占用名次区间”的平均值
	actualRankMap := make(map[string]float64, len(participants))
	sortedByRank := make([]Participant, len(participants))
	copy(sortedByRank, participants)
	sort.Slice(sortedByRank, func(i, j int) bool {
		return sortedByRank[i].Rank < sortedByRank[j].Rank
	})

	pos := 1
	for i := 0; i < len(sortedByRank); {
		rank := sortedByRank[i].Rank
		if rank <= 0 {
			rank = 1
		}
		j := i + 1
		for j < len(sortedByRank) {
			nextRank := sortedByRank[j].Rank
			if nextRank <= 0 {
				nextRank = 1
			}
			if nextRank != rank {
				break
			}
			j++
		}
		startPos := pos
		endPos := pos + (j - i) - 1
		avgPos := float64(startPos+endPos) / 2.0
		for k := i; k < j; k++ {
			actualRankMap[sortedByRank[k].StudentID] = avgPos
		}
		pos += (j - i)
		i = j
	}

	var changes []RatingChange

	for _, p := range participants {
		actualRank := actualRankMap[p.StudentID]
		expectedRank := expectedRankMap[p.StudentID]

		if actualRank <= 0 {
			actualRank = 1
		}

		// 排名百分位 (0, 1]，排名越高值越小
		percentile := actualRank / n

		// === 1. 参与奖励 ===
		participationBonus := config.ParticipationBonus

		// === 2. 排名梯度奖励 ===
		// 使用可调曲线: bonus = max × (1 - percentile)^curve
		// curve > 1 时曲线更平缓，更多人获得较高分数
		// curve < 1 时曲线更陡峭，主要奖励头部选手
		rankBonus := 0.0
		if config.RankBonusMax > 0 {
			curveFactor := config.RankBonusCurve
			if curveFactor <= 0 {
				curveFactor = 1.0
			}
			// 1 - percentile: 第1名接近1，最后一名接近0
			rankBonus = config.RankBonusMax * math.Pow(1-percentile, 1/curveFactor)
		}

		// === 3. 技能差异奖励（原Elo组件）===
		seed := n / actualRank
		performance := baseRating + k*math.Log(seed)

		expectedSeed := n / expectedRank
		expectedPerformance := baseRating + k*math.Log(expectedSeed)

		skillDelta := performance - expectedPerformance

		// === 4. Top奖励 ===
		topBonus := 0.0
		if config.TopBonusThreshold > 0 && percentile <= config.TopBonusThreshold {
			topBonus = k * config.TopBonusMultiplier * (config.TopBonusThreshold - percentile) / config.TopBonusThreshold
		}

		// === 汇总计算 ===
		// delta = (参与分 + 排名分 + 技能差异分 + Top奖励) × 阻尼系数
		delta := (participationBonus + rankBonus + skillDelta + topBonus) * config.GrowthInertia

		newRating := p.CurrentRating + delta

		changes = append(changes, RatingChange{
			StudentID:    p.StudentID,
			Performance:  math.Round(performance*100) / 100,
			RatingBefore: p.CurrentRating,
			RatingAfter:  math.Round(newRating*100) / 100,
			Delta:        math.Round(delta*100) / 100,
		})
	}

	return changes, nil
}
