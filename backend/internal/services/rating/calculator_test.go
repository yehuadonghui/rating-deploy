package rating

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"rating-system/internal/models"
)

func TestEloMMRCalculator_Calculate(t *testing.T) {
	calculator := NewEloMMRCalculator()
	config := DefaultConfig()
	ctx := context.Background()

	t.Run("超出期望获得涨分", func(t *testing.T) {
		contest := &models.Contest{ID: 1, Name: "测试比赛", Weight: 1.0}
		// 低分选手超常发挥，排名第一
		participants := []Participant{
			{StudentID: "001", CurrentRating: 0, Rank: 1, Solved: 10},    // 0分选手拿第一
			{StudentID: "002", CurrentRating: 100, Rank: 2, Solved: 9},   // 100分选手拿第二
			{StudentID: "003", CurrentRating: 200, Rank: 3, Solved: 8},   // 200分选手拿第三
		}

		changes, err := calculator.Calculate(ctx, contest, participants, config)
		require.NoError(t, err)
		assert.Len(t, changes, 3)

		// 第一名（低分选手超常发挥）应该涨分
		assert.Greater(t, changes[0].Delta, 0.0, "超出期望应该涨分")
		// 所有选手的 delta >= 0（永不下降）
		for _, c := range changes {
			assert.GreaterOrEqual(t, c.Delta, 0.0)
		}
	})

	t.Run("永不下降机制", func(t *testing.T) {
		contest := &models.Contest{ID: 1, Name: "测试比赛", Weight: 1.0}
		// 高分选手表现不佳
		participants := []Participant{
			{StudentID: "001", CurrentRating: 1000, Rank: 3, Solved: 3}, // 高分选手排第三
			{StudentID: "002", CurrentRating: 0, Rank: 1, Solved: 5},
			{StudentID: "003", CurrentRating: 0, Rank: 2, Solved: 4},
		}

		changes, err := calculator.Calculate(ctx, contest, participants, config)
		require.NoError(t, err)

		// 高分选手 delta 应该 >= 0（永不下降）
		for _, c := range changes {
			assert.GreaterOrEqual(t, c.Delta, 0.0, "Delta should never be negative")
		}
	})

	t.Run("比赛权重影响", func(t *testing.T) {
		// 有Rating差异的场景
		participants := []Participant{
			{StudentID: "001", CurrentRating: 0, Rank: 1, Solved: 10},   // 低分选手拿第一
			{StudentID: "002", CurrentRating: 100, Rank: 2, Solved: 9},
			{StudentID: "003", CurrentRating: 200, Rank: 3, Solved: 8},
		}

		// 权重 1.0
		contest1 := &models.Contest{ID: 1, Name: "普通比赛", Weight: 1.0}
		changes1, _ := calculator.Calculate(ctx, contest1, participants, config)

		// 权重 2.0
		contest2 := &models.Contest{ID: 2, Name: "重要比赛", Weight: 2.0}
		changes2, _ := calculator.Calculate(ctx, contest2, participants, config)

		// 高权重比赛涨分应该更多
		assert.Greater(t, changes2[0].Delta, changes1[0].Delta)
	})

	t.Run("空参赛者列表", func(t *testing.T) {
		contest := &models.Contest{ID: 1, Name: "测试比赛", Weight: 1.0}
		changes, err := calculator.Calculate(ctx, contest, []Participant{}, config)
		require.NoError(t, err)
		assert.Nil(t, changes)
	})

	t.Run("单人参赛", func(t *testing.T) {
		contest := &models.Contest{ID: 1, Name: "测试比赛", Weight: 1.0}
		participants := []Participant{
			{StudentID: "001", CurrentRating: 0, Rank: 1, Solved: 5},
		}

		changes, err := calculator.Calculate(ctx, contest, participants, config)
		require.NoError(t, err)
		assert.Len(t, changes, 1)
		assert.GreaterOrEqual(t, changes[0].Delta, 0.0)
	})
}

func TestEloMMRCalculator_Name(t *testing.T) {
	calculator := NewEloMMRCalculator()
	assert.Equal(t, "Modified Elo-MMR", calculator.Name())
}
