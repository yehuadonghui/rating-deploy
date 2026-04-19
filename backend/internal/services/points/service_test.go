package points

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"rating-system/internal/models"
)

func TestGenerateContestRecords_AppliesPrizeTiersAndParticipation(t *testing.T) {
	db := newPointsTestDB(t)
	service := NewService(db)

	contest := models.Contest{
		ID:               1,
		Name:             "Foundation Exam 1",
		Date:             time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		ParticipantCount: 100,
		RewardType:       models.ContestRewardTypeFoundationExam,
	}
	results := []models.Result{
		{ContestID: 1, StudentID: "001", Rank: 1},
		{ContestID: 1, StudentID: "002", Rank: 2},
		{ContestID: 1, StudentID: "003", Rank: 3},
		{ContestID: 1, StudentID: "004", Rank: 4},
	}

	records, err := service.GenerateContestRecords(&contest, results)
	require.NoError(t, err)
	require.Len(t, records, 4)

	assert.Equal(t, 100, records[0].Points)
	assert.Equal(t, 80, records[1].Points)
	assert.Equal(t, 50, records[2].Points)
	assert.Equal(t, 5, records[3].Points)
	assert.Equal(t, models.PointCategoryContestAward, records[0].Category)
	assert.Equal(t, models.PointSourceAuto, records[0].Source)
}

func TestGenerateContestRecords_NoGuaranteedTopAwardsForSmallContests(t *testing.T) {
	db := newPointsTestDB(t)
	service := NewService(db)

	contest := models.Contest{
		ID:               2,
		Name:             "Tiny Monthly",
		Date:             time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC),
		ParticipantCount: 20,
		RewardType:       models.ContestRewardTypeBrandMonthly,
	}
	results := []models.Result{
		{ContestID: 2, StudentID: "001", Rank: 1},
		{ContestID: 2, StudentID: "002", Rank: 2},
	}

	records, err := service.GenerateContestRecords(&contest, results)
	require.NoError(t, err)
	require.Len(t, records, 2)

	assert.Equal(t, 2, records[0].Points)
	assert.Equal(t, 2, records[1].Points)
}

func TestRecomputeStudentTotals_CombinesAutoAndManualRecords(t *testing.T) {
	db := newPointsTestDB(t)
	service := NewService(db)

	students := []models.Student{
		{StudentID: "001", Name: "Alice"},
		{StudentID: "002", Name: "Bob"},
	}
	require.NoError(t, db.Create(&students).Error)
	require.NoError(t, db.Create(&[]models.PointRecord{
		{
			StudentID: "001",
			ContestID: uintPtr(1),
			Category:  models.PointCategoryContestAward,
			Source:    models.PointSourceAuto,
			Points:    100,
			Title:     "Foundation first prize",
		},
		{
			StudentID: "001",
			Category:  models.PointCategoryProgressAward,
			Source:    models.PointSourceManual,
			Points:    50,
			Title:     "Progress award",
		},
		{
			StudentID: "002",
			Category:  models.PointCategoryOrganizerReward,
			Source:    models.PointSourceManual,
			Points:    20,
			Title:     "Organizer reward",
		},
	}).Error)

	require.NoError(t, service.RecomputeStudentTotals([]string{"001", "002"}))

	var updated []models.Student
	require.NoError(t, db.Order("student_id ASC").Find(&updated).Error)
	require.Len(t, updated, 2)

	assert.Equal(t, 150, updated[0].RedeemPoints)
	assert.Equal(t, 20, updated[1].RedeemPoints)
}

func TestListPointRecords_FiltersAndOrdersResults(t *testing.T) {
	db := newPointsTestDB(t)
	service := NewService(db)

	contestID := uint(3)
	require.NoError(t, db.Create(&[]models.PointRecord{
		{
			StudentID: "001",
			ContestID: &contestID,
			Category:  models.PointCategoryContestAward,
			Source:    models.PointSourceAuto,
			Points:    100,
			Title:     "A",
		},
		{
			StudentID: "001",
			Category:  models.PointCategoryProgressAward,
			Source:    models.PointSourceManual,
			Points:    50,
			Title:     "B",
		},
		{
			StudentID: "002",
			Category:  models.PointCategoryOrganizerReward,
			Source:    models.PointSourceManual,
			Points:    20,
			Title:     "C",
		},
	}).Error)

	records, total, err := service.ListPointRecords(ListQuery{
		StudentID: "001",
		Source:    models.PointSourceManual,
		Page:      1,
		Size:      10,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, records, 1)
	assert.Equal(t, "B", records[0].Title)
}

func TestListStudentPointRecords_ReturnsStudentRecordsOnly(t *testing.T) {
	db := newPointsTestDB(t)
	service := NewService(db)

	require.NoError(t, db.Create(&[]models.PointRecord{
		{
			StudentID: "001",
			Category:  models.PointCategoryContestAward,
			Source:    models.PointSourceAuto,
			Points:    100,
			Title:     "Contest",
		},
		{
			StudentID: "002",
			Category:  models.PointCategoryProgressAward,
			Source:    models.PointSourceManual,
			Points:    50,
			Title:     "Progress",
		},
	}).Error)

	records, err := service.ListStudentPointRecords("001")
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, "001", records[0].StudentID)
	assert.Equal(t, "Contest", records[0].Title)
}

func newPointsTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s-%d?mode=memory&cache=shared", t.Name(), time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.Student{},
		&models.Contest{},
		&models.Result{},
		&models.PointRecord{},
	))

	return db
}

func uintPtr(v uint) *uint {
	return &v
}
