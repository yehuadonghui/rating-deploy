package points

import (
	"fmt"
	"math"
	"sort"

	"gorm.io/gorm"

	"rating-system/internal/models"
)

type Service struct {
	db *gorm.DB
}

type ListQuery struct {
	StudentID string
	ContestID uint
	Category  models.PointCategory
	Source    models.PointSource
	Page      int
	Size      int
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

type prizeTable struct {
	first         int
	second        int
	third         int
	participation int
}

func (s *Service) GenerateContestRecords(contest *models.Contest, results []models.Result) ([]models.PointRecord, error) {
	if contest == nil || contest.RewardType == models.ContestRewardTypeNone {
		return nil, nil
	}

	table, ok := rewardTable(contest.RewardType)
	if !ok {
		return nil, fmt.Errorf("unsupported reward type: %s", contest.RewardType)
	}
	if len(results) == 0 {
		return nil, nil
	}

	sorted := make([]models.Result, len(results))
	copy(sorted, results)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Rank == sorted[j].Rank {
			return sorted[i].StudentID < sorted[j].StudentID
		}
		return sorted[i].Rank < sorted[j].Rank
	})

	participantCount := contest.ParticipantCount
	if participantCount <= 0 {
		participantCount = len(sorted)
	}

	firstLimit := int(math.Floor(float64(participantCount) * 0.01))
	secondLimit := int(math.Floor(float64(participantCount) * 0.02))
	thirdLimit := int(math.Floor(float64(participantCount) * 0.03))

	records := make([]models.PointRecord, 0, len(sorted))
	for _, result := range sorted {
		points := table.participation
		title := "参与奖"

		switch {
		case firstLimit > 0 && result.Rank <= firstLimit:
			points = table.first
			title = "一等奖"
		case secondLimit > 0 && result.Rank <= secondLimit:
			points = table.second
			title = "二等奖"
		case thirdLimit > 0 && result.Rank <= thirdLimit:
			points = table.third
			title = "三等奖"
		}

		contestID := contest.ID
		records = append(records, models.PointRecord{
			StudentID: result.StudentID,
			ContestID: &contestID,
			Category:  models.PointCategoryContestAward,
			Source:    models.PointSourceAuto,
			Points:    points,
			Title:     fmt.Sprintf("%s %s", contest.Name, title),
		})
	}

	return records, nil
}

func (s *Service) ReplaceContestAutoRecords(contest *models.Contest) error {
	if contest == nil {
		return nil
	}

	var existing []models.PointRecord
	if err := s.db.Where("contest_id = ? AND source = ?", contest.ID, models.PointSourceAuto).Find(&existing).Error; err != nil {
		return err
	}

	studentIDs := make([]string, 0, len(existing))
	for _, record := range existing {
		studentIDs = append(studentIDs, record.StudentID)
	}

	if err := s.db.Where("contest_id = ? AND source = ?", contest.ID, models.PointSourceAuto).Delete(&models.PointRecord{}).Error; err != nil {
		return err
	}

	if contest.RewardType != models.ContestRewardTypeNone {
		var results []models.Result
		if err := s.db.Where("contest_id = ?", contest.ID).Find(&results).Error; err != nil {
			return err
		}
		records, err := s.GenerateContestRecords(contest, results)
		if err != nil {
			return err
		}
		if len(records) > 0 {
			if err := s.db.Create(&records).Error; err != nil {
				return err
			}
			for _, record := range records {
				studentIDs = append(studentIDs, record.StudentID)
			}
		}
	}

	return s.RecomputeStudentTotals(studentIDs)
}

func (s *Service) DeleteContestAutoRecords(contestID uint) error {
	var existing []models.PointRecord
	if err := s.db.Where("contest_id = ? AND source = ?", contestID, models.PointSourceAuto).Find(&existing).Error; err != nil {
		return err
	}

	studentIDs := make([]string, 0, len(existing))
	for _, record := range existing {
		studentIDs = append(studentIDs, record.StudentID)
	}

	if err := s.db.Where("contest_id = ? AND source = ?", contestID, models.PointSourceAuto).Delete(&models.PointRecord{}).Error; err != nil {
		return err
	}

	return s.RecomputeStudentTotals(studentIDs)
}

func (s *Service) CreateManualRecord(record *models.PointRecord) error {
	if record == nil {
		return nil
	}
	record.Source = models.PointSourceManual
	if err := s.db.Create(record).Error; err != nil {
		return err
	}
	return s.RecomputeStudentTotals([]string{record.StudentID})
}

func (s *Service) RecomputeStudentTotals(studentIDs []string) error {
	unique := uniqueStudentIDs(studentIDs)
	for _, studentID := range unique {
		var total struct {
			Sum int
		}
		if err := s.db.Model(&models.PointRecord{}).
			Select("COALESCE(SUM(points), 0) AS sum").
			Where("student_id = ?", studentID).
			Scan(&total).Error; err != nil {
			return err
		}
		if err := s.db.Model(&models.Student{}).
			Where("student_id = ?", studentID).
			Update("redeem_points", total.Sum).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) ListPointRecords(query ListQuery) ([]models.PointRecord, int64, error) {
	page := query.Page
	if page <= 0 {
		page = 1
	}
	size := query.Size
	if size <= 0 {
		size = 20
	}

	dbQuery := s.db.Model(&models.PointRecord{})
	if query.StudentID != "" {
		dbQuery = dbQuery.Where("student_id = ?", query.StudentID)
	}
	if query.ContestID > 0 {
		dbQuery = dbQuery.Where("contest_id = ?", query.ContestID)
	}
	if query.Category != "" {
		dbQuery = dbQuery.Where("category = ?", query.Category)
	}
	if query.Source != "" {
		dbQuery = dbQuery.Where("source = ?", query.Source)
	}

	var total int64
	if err := dbQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var records []models.PointRecord
	if err := dbQuery.
		Preload("Student").
		Preload("Contest").
		Order("created_at DESC, id DESC").
		Offset((page - 1) * size).
		Limit(size).
		Find(&records).Error; err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

func (s *Service) ListStudentPointRecords(studentID string) ([]models.PointRecord, error) {
	var records []models.PointRecord
	err := s.db.
		Preload("Contest").
		Where("student_id = ?", studentID).
		Order("created_at DESC, id DESC").
		Find(&records).Error
	if err != nil {
		return nil, err
	}
	return records, nil
}

func rewardTable(rewardType models.ContestRewardType) (prizeTable, bool) {
	switch rewardType {
	case models.ContestRewardTypeFoundationExam:
		return prizeTable{first: 100, second: 80, third: 50, participation: 5}, true
	case models.ContestRewardTypeBrandMonthly:
		return prizeTable{first: 50, second: 30, third: 20, participation: 2}, true
	case models.ContestRewardTypeBrandFinal:
		return prizeTable{first: 100, second: 80, third: 50, participation: 5}, true
	default:
		return prizeTable{}, false
	}
}

func uniqueStudentIDs(studentIDs []string) []string {
	seen := make(map[string]struct{}, len(studentIDs))
	unique := make([]string, 0, len(studentIDs))
	for _, studentID := range studentIDs {
		if studentID == "" {
			continue
		}
		if _, ok := seen[studentID]; ok {
			continue
		}
		seen[studentID] = struct{}{}
		unique = append(unique, studentID)
	}
	return unique
}
