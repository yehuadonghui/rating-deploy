package rating

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"rating-system/internal/models"
)

func TestReplayAll_AppliesHistoryDecayToOlderContests(t *testing.T) {
	withDecay := recordReplayWeights(t, 0.5)
	withoutDecay := recordReplayWeights(t, 1.0)

	assert.Equal(t, []float64{0.5, 1.0}, withDecay)
	assert.Equal(t, []float64{1.0, 1.0}, withoutDecay)
}

func TestPreviewAll_MatchesReplayAllWhenRatingsDrop(t *testing.T) {
	config := RatingConfig{
		KFactor:            200,
		GrowthInertia:      1,
		HistoryDecay:       1,
		InitialRating:      0,
		TopBonusThreshold:  0,
		TopBonusMultiplier: 0,
		ParticipationBonus: 0,
		RankBonusMax:       0,
		RankBonusCurve:     1,
	}

	db := newReplayTestDB(t)
	seedRatingDropFixture(t, db)

	engine := NewReplayEngine(db, NewEloMMRCalculator())
	engine.SetConfig(config)

	preview, err := engine.PreviewAll(context.Background())
	require.NoError(t, err)

	require.NoError(t, engine.ReplayAll(context.Background(), nil))

	var students []models.Student
	require.NoError(t, db.Order("student_id ASC").Find(&students).Error)

	previewByID := make(map[string]PreviewStudent, len(preview))
	for _, item := range preview {
		previewByID[item.StudentID] = item
	}

	require.Len(t, previewByID, len(students))
	assert.Less(t, students[0].CurrentRating, students[0].MaxRating)

	for _, student := range students {
		item, ok := previewByID[student.StudentID]
		require.True(t, ok, "missing preview for %s", student.StudentID)
		assert.InDelta(t, student.CurrentRating, item.CurrentRating, 0.01)
		assert.InDelta(t, student.MaxRating, item.MaxRating, 0.01)
		assert.Equal(t, student.MatchCount, item.MatchCount)
	}
}

func recordReplayWeights(t *testing.T, historyDecay float64) []float64 {
	t.Helper()

	db := newReplayTestDB(t)
	seedHistoryDecayFixture(t, db)

	calculator := &recordingCalculator{}
	engine := NewReplayEngine(db, calculator)
	engine.SetConfig(RatingConfig{
		KFactor:            100,
		GrowthInertia:      1,
		HistoryDecay:       historyDecay,
		InitialRating:      0,
		TopBonusThreshold:  0,
		TopBonusMultiplier: 0,
		ParticipationBonus: 0,
		RankBonusMax:       0,
		RankBonusCurve:     1,
	})

	require.NoError(t, engine.ReplayAll(context.Background(), nil))
	return calculator.weights
}

func newReplayTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s-%d?mode=memory&cache=shared", t.Name(), time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.Student{},
		&models.Contest{},
		&models.Result{},
		&models.SystemConfig{},
		&models.TierConfig{},
	))

	return db
}

func seedHistoryDecayFixture(t *testing.T, db *gorm.DB) {
	t.Helper()

	students := []models.Student{
		{StudentID: "001", Name: "Alice"},
		{StudentID: "002", Name: "Bob"},
	}
	contests := []models.Contest{
		{ID: 1, Name: "Old", Date: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Weight: 1},
		{ID: 2, Name: "New", Date: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC), Weight: 1},
	}
	results := []models.Result{
		{ContestID: 1, StudentID: "001", Rank: 1, Solved: 5},
		{ContestID: 1, StudentID: "002", Rank: 2, Solved: 4},
		{ContestID: 2, StudentID: "001", Rank: 2, Solved: 4},
		{ContestID: 2, StudentID: "002", Rank: 1, Solved: 5},
	}

	require.NoError(t, db.Create(&students).Error)
	require.NoError(t, db.Create(&contests).Error)
	require.NoError(t, db.Create(&results).Error)
}

func seedRatingDropFixture(t *testing.T, db *gorm.DB) {
	t.Helper()

	students := []models.Student{
		{StudentID: "001", Name: "Alice"},
		{StudentID: "002", Name: "Bob"},
	}
	contests := []models.Contest{
		{ID: 1, Name: "Round 1", Date: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Weight: 1},
		{ID: 2, Name: "Round 2", Date: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), Weight: 1},
	}
	results := []models.Result{
		{ContestID: 1, StudentID: "001", Rank: 1, Solved: 5},
		{ContestID: 1, StudentID: "002", Rank: 2, Solved: 4},
		{ContestID: 2, StudentID: "001", Rank: 2, Solved: 1},
		{ContestID: 2, StudentID: "002", Rank: 1, Solved: 5},
	}

	require.NoError(t, db.Create(&students).Error)
	require.NoError(t, db.Create(&contests).Error)
	require.NoError(t, db.Create(&results).Error)
}

type recordingCalculator struct {
	weights []float64
}

func (c *recordingCalculator) Calculate(
	_ context.Context,
	contest *models.Contest,
	participants []Participant,
	_ RatingConfig,
) ([]RatingChange, error) {
	c.weights = append(c.weights, contest.Weight)

	changes := make([]RatingChange, 0, len(participants))
	for _, participant := range participants {
		changes = append(changes, RatingChange{
			StudentID:    participant.StudentID,
			RatingBefore: participant.CurrentRating,
			RatingAfter:  participant.CurrentRating,
		})
	}

	return changes, nil
}

func (c *recordingCalculator) Name() string {
	return "recording"
}
