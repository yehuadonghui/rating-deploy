package rating

import (
	"context"
	"log"
	"math"
	"strconv"
	"sync"

	"gorm.io/gorm"

	"rating-system/internal/models"
)

// ReplayProgress tracks replay status for polling.
type ReplayProgress struct {
	CurrentContest int    `json:"current_contest"`
	TotalContests  int    `json:"total_contests"`
	ContestName    string `json:"contest_name"`
	Status         string `json:"status"` // processing, completed, error
}

// ReplayEngine recalculates all historical ratings.
type ReplayEngine struct {
	db         *gorm.DB
	calculator RatingCalculator
	config     RatingConfig
	mu         sync.Mutex
	isRunning  bool
	progress   ReplayProgress
}

// NewReplayEngine creates a replay engine with default config.
func NewReplayEngine(db *gorm.DB, calculator RatingCalculator) *ReplayEngine {
	return &ReplayEngine{
		db:         db,
		calculator: calculator,
		config:     DefaultConfig(),
	}
}

// SetConfig overrides the replay config.
func (e *ReplayEngine) SetConfig(config RatingConfig) {
	e.config = config
}

// LoadConfigFromDB loads persisted replay config.
func (e *ReplayEngine) LoadConfigFromDB() error {
	var configs []models.SystemConfig
	if err := e.db.Find(&configs).Error; err != nil {
		return err
	}

	for _, cfg := range configs {
		switch cfg.Key {
		case "k_factor":
			if v, err := strconv.ParseFloat(cfg.Value, 64); err == nil {
				e.config.KFactor = v
			}
		case "growth_inertia":
			if v, err := strconv.ParseFloat(cfg.Value, 64); err == nil {
				e.config.GrowthInertia = v
			}
		case "history_decay":
			if v, err := strconv.ParseFloat(cfg.Value, 64); err == nil {
				e.config.HistoryDecay = v
			}
		case "initial_rating":
			if v, err := strconv.ParseFloat(cfg.Value, 64); err == nil {
				e.config.InitialRating = v
			}
		case "top_bonus_threshold":
			if v, err := strconv.ParseFloat(cfg.Value, 64); err == nil {
				e.config.TopBonusThreshold = v
			}
		case "top_bonus_multiplier":
			if v, err := strconv.ParseFloat(cfg.Value, 64); err == nil {
				e.config.TopBonusMultiplier = v
			}
		}
	}

	return nil
}

// IsRunning reports whether a replay is active.
func (e *ReplayEngine) IsRunning() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.isRunning
}

// Progress returns the latest replay progress snapshot.
func (e *ReplayEngine) Progress() ReplayProgress {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.progress
}

func (e *ReplayEngine) setProgress(p ReplayProgress) {
	e.mu.Lock()
	e.progress = p
	e.mu.Unlock()
}

// ReplayAll recalculates ratings for every contest in chronological order.
func (e *ReplayEngine) ReplayAll(ctx context.Context, progressCh chan<- ReplayProgress) error {
	e.mu.Lock()
	if e.isRunning {
		e.mu.Unlock()
		return ErrReplayInProgress
	}
	e.isRunning = true
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		e.isRunning = false
		e.mu.Unlock()
	}()

	if err := e.LoadConfigFromDB(); err != nil {
		return err
	}

	return e.db.Transaction(func(tx *gorm.DB) error {
		log.Println("resetting all student ratings")
		if err := tx.Model(&models.Student{}).Where("1=1").Updates(map[string]interface{}{
			"current_rating": 0,
			"max_rating":     0,
			"match_count":    0,
		}).Error; err != nil {
			return err
		}

		var contests []models.Contest
		if err := tx.Order("date ASC").Find(&contests).Error; err != nil {
			return err
		}

		if len(contests) == 0 {
			completed := ReplayProgress{Status: "completed"}
			e.setProgress(completed)
			if progressCh != nil {
				progressCh <- completed
			}
			return nil
		}

		for i, contest := range contests {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			current := ReplayProgress{
				CurrentContest: i + 1,
				TotalContests:  len(contests),
				ContestName:    contest.Name,
				Status:         "processing",
			}
			e.setProgress(current)
			if progressCh != nil {
				progressCh <- current
			}

			effectiveContest := contest
			effectiveContest.Weight = e.decayedContestWeight(contest.Weight, i, len(contests))

			if err := e.processContest(tx, &effectiveContest); err != nil {
				log.Printf("process contest %s failed: %v", contest.Name, err)
				e.setProgress(ReplayProgress{
					CurrentContest: i + 1,
					TotalContests:  len(contests),
					ContestName:    contest.Name,
					Status:         "error",
				})
				return err
			}
		}

		completed := ReplayProgress{
			CurrentContest: len(contests),
			TotalContests:  len(contests),
			Status:         "completed",
		}
		e.setProgress(completed)
		if progressCh != nil {
			progressCh <- completed
		}

		return nil
	})
}

func (e *ReplayEngine) processContest(tx *gorm.DB, contest *models.Contest) error {
	var results []models.Result
	if err := tx.Where("contest_id = ?", contest.ID).Find(&results).Error; err != nil {
		return err
	}
	if len(results) == 0 {
		return nil
	}

	studentIDs := make([]string, len(results))
	for i, r := range results {
		studentIDs[i] = r.StudentID
	}

	var students []models.Student
	if err := tx.Where("student_id IN ?", studentIDs).Find(&students).Error; err != nil {
		return err
	}

	studentMap := make(map[string]*models.Student, len(students))
	for i := range students {
		studentMap[students[i].StudentID] = &students[i]
	}

	var participants []Participant
	for _, r := range results {
		student := studentMap[r.StudentID]
		if student == nil {
			continue
		}
		participants = append(participants, Participant{
			StudentID:     r.StudentID,
			CurrentRating: student.CurrentRating,
			Rank:          r.Rank,
			Solved:        r.Solved,
		})
	}

	changes, err := e.calculator.Calculate(context.Background(), contest, participants, e.config)
	if err != nil {
		return err
	}

	changeMap := make(map[string]RatingChange, len(changes))
	for _, change := range changes {
		changeMap[change.StudentID] = change
	}

	for _, result := range results {
		change, ok := changeMap[result.StudentID]
		if !ok {
			continue
		}

		if err := tx.Model(&result).Updates(map[string]interface{}{
			"performance":   change.Performance,
			"rating_before": change.RatingBefore,
			"rating_after":  change.RatingAfter,
			"delta":         change.Delta,
		}).Error; err != nil {
			return err
		}

		student := studentMap[result.StudentID]
		if student == nil {
			continue
		}

		student.UpdateRating(change.RatingAfter)
		student.MatchCount++
		if err := tx.Model(student).Updates(map[string]interface{}{
			"current_rating": student.CurrentRating,
			"max_rating":     student.MaxRating,
			"match_count":    student.MatchCount,
		}).Error; err != nil {
			return err
		}
	}

	return nil
}

// ErrReplayInProgress is returned when another replay is already running.
var ErrReplayInProgress = &ReplayError{Message: "replay is already running"}

// ReplayError represents replay-specific failures.
type ReplayError struct {
	Message string
}

func (e *ReplayError) Error() string {
	return e.Message
}

// PreviewStudent is the in-memory replay result for one student.
type PreviewStudent struct {
	StudentID     string  `json:"student_id"`
	Name          string  `json:"name"`
	Class         string  `json:"class"`
	CurrentRating float64 `json:"current_rating"`
	MaxRating     float64 `json:"max_rating"`
	MatchCount    int     `json:"match_count"`
}

// PreviewAll simulates ReplayAll without persisting changes.
func (e *ReplayEngine) PreviewAll(ctx context.Context) ([]PreviewStudent, error) {
	studentRatings := make(map[string]*PreviewStudent)

	var students []models.Student
	if err := e.db.Find(&students).Error; err != nil {
		return nil, err
	}

	for _, s := range students {
		studentRatings[s.StudentID] = &PreviewStudent{
			StudentID:     s.StudentID,
			Name:          s.Name,
			Class:         s.Class,
			CurrentRating: 0,
			MaxRating:     0,
			MatchCount:    0,
		}
	}

	var contests []models.Contest
	if err := e.db.Order("date ASC").Find(&contests).Error; err != nil {
		return nil, err
	}

	for i, contest := range contests {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		effectiveContest := contest
		effectiveContest.Weight = e.decayedContestWeight(contest.Weight, i, len(contests))

		var results []models.Result
		if err := e.db.Where("contest_id = ?", contest.ID).Find(&results).Error; err != nil {
			return nil, err
		}
		if len(results) == 0 {
			continue
		}

		var participants []Participant
		for _, r := range results {
			student := studentRatings[r.StudentID]
			if student == nil {
				continue
			}
			participants = append(participants, Participant{
				StudentID:     r.StudentID,
				CurrentRating: student.CurrentRating,
				Rank:          r.Rank,
				Solved:        r.Solved,
			})
		}

		changes, err := e.calculator.Calculate(ctx, &effectiveContest, participants, e.config)
		if err != nil {
			return nil, err
		}

		for _, change := range changes {
			student := studentRatings[change.StudentID]
			if student == nil {
				continue
			}

			student.CurrentRating = change.RatingAfter
			if change.RatingAfter > student.MaxRating {
				student.MaxRating = change.RatingAfter
			}
			student.MatchCount++
		}
	}

	var result []PreviewStudent
	for _, student := range studentRatings {
		if student.MatchCount > 0 {
			result = append(result, *student)
		}
	}

	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].CurrentRating > result[i].CurrentRating {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result, nil
}

func (e *ReplayEngine) decayedContestWeight(baseWeight float64, contestIndex int, totalContests int) float64 {
	if baseWeight <= 0 {
		baseWeight = 1
	}
	if totalContests <= 1 {
		return baseWeight
	}

	decay := e.config.HistoryDecay
	if decay <= 0 {
		return baseWeight
	}

	remainingContests := totalContests - contestIndex - 1
	if remainingContests <= 0 {
		return baseWeight
	}

	return baseWeight * math.Pow(decay, float64(remainingContests))
}
