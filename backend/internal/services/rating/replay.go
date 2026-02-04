package rating

import (
	"context"
	"log"
	"strconv"
	"sync"

	"gorm.io/gorm"

	"rating-system/internal/models"
)

// ReplayProgress 重算进度
type ReplayProgress struct {
	CurrentContest int    `json:"current_contest"`
	TotalContests  int    `json:"total_contests"`
	ContestName    string `json:"contest_name"`
	Status         string `json:"status"` // processing, completed, error
}

// ReplayEngine 重算引擎
type ReplayEngine struct {
	db         *gorm.DB
	calculator RatingCalculator
	config     RatingConfig
	mu         sync.Mutex
	isRunning  bool
	progress   ReplayProgress
}

// NewReplayEngine 创建重算引擎
func NewReplayEngine(db *gorm.DB, calculator RatingCalculator) *ReplayEngine {
	return &ReplayEngine{
		db:         db,
		calculator: calculator,
		config:     DefaultConfig(),
	}
}

// SetConfig 设置配置
func (e *ReplayEngine) SetConfig(config RatingConfig) {
	e.config = config
}

// LoadConfigFromDB 从数据库加载配置
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

// IsRunning 检查是否正在运行
func (e *ReplayEngine) IsRunning() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.isRunning
}

// Progress 获取当前进度快照
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

// ReplayAll 全量重算
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

	// 加载最新配置
	if err := e.LoadConfigFromDB(); err != nil {
		return err
	}

	return e.db.Transaction(func(tx *gorm.DB) error {
		// 1. 重置所有学生 Rating
		log.Println("重置所有学生 Rating...")
		if err := tx.Model(&models.Student{}).Where("1=1").Updates(map[string]interface{}{
			"current_rating": 0,
			"max_rating":     0,
			"match_count":    0,
		}).Error; err != nil {
			return err
		}

		// 2. 加载所有比赛（按时间排序）
		var contests []models.Contest
		if err := tx.Order("date ASC").Find(&contests).Error; err != nil {
			return err
		}

		if len(contests) == 0 {
			e.setProgress(ReplayProgress{
				Status: "completed",
			})
			if progressCh != nil {
				progressCh <- ReplayProgress{
					Status: "completed",
				}
			}
			return nil
		}

		// 3. 逐场处理
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

			if err := e.processContest(tx, &contest); err != nil {
				log.Printf("处理比赛 %s 失败: %v", contest.Name, err)
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

// processContest 处理单场比赛
func (e *ReplayEngine) processContest(tx *gorm.DB, contest *models.Contest) error {
	// 加载该比赛的所有结果
	var results []models.Result
	if err := tx.Where("contest_id = ?", contest.ID).Find(&results).Error; err != nil {
		return err
	}

	if len(results) == 0 {
		return nil
	}

	// 获取参赛者当前 Rating
	studentIDs := make([]string, len(results))
	for i, r := range results {
		studentIDs[i] = r.StudentID
	}

	var students []models.Student
	if err := tx.Where("student_id IN ?", studentIDs).Find(&students).Error; err != nil {
		return err
	}

	studentMap := make(map[string]*models.Student)
	for i := range students {
		studentMap[students[i].StudentID] = &students[i]
	}

	// 构建参赛者列表
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

	// 计算 Rating 变化
	changes, err := e.calculator.Calculate(context.Background(), contest, participants, e.config)
	if err != nil {
		return err
	}

	// 更新结果和学生 Rating
	changeMap := make(map[string]RatingChange)
	for _, c := range changes {
		changeMap[c.StudentID] = c
	}

	for _, r := range results {
		change, ok := changeMap[r.StudentID]
		if !ok {
			continue
		}

		// 更新 Result
		if err := tx.Model(&r).Updates(map[string]interface{}{
			"performance":   change.Performance,
			"rating_before": change.RatingBefore,
			"rating_after":  change.RatingAfter,
			"delta":         change.Delta,
		}).Error; err != nil {
			return err
		}

		// 更新 Student
		student := studentMap[r.StudentID]
		if student != nil {
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
	}

	return nil
}

// ErrReplayInProgress 重算正在进行中
var ErrReplayInProgress = &ReplayError{Message: "重算正在进行中，请稍后再试"}

// ReplayError 重算错误
type ReplayError struct {
	Message string
}

func (e *ReplayError) Error() string {
	return e.Message
}

// PreviewStudent 预览学生结果
type PreviewStudent struct {
	StudentID     string  `json:"student_id"`
	Name          string  `json:"name"`
	Class         string  `json:"class"`
	CurrentRating float64 `json:"current_rating"`
	MaxRating     float64 `json:"max_rating"`
	MatchCount    int     `json:"match_count"`
}

// PreviewAll 预览全量重算结果（不保存到数据库）
func (e *ReplayEngine) PreviewAll(ctx context.Context) ([]PreviewStudent, error) {
	// 内存中模拟学生状态
	studentRatings := make(map[string]*PreviewStudent)

	// 加载所有学生
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

	// 加载所有比赛（按时间排序）
	var contests []models.Contest
	if err := e.db.Order("date ASC").Find(&contests).Error; err != nil {
		return nil, err
	}

	// 逐场处理
	for _, contest := range contests {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// 加载该比赛的所有结果
		var results []models.Result
		if err := e.db.Where("contest_id = ?", contest.ID).Find(&results).Error; err != nil {
			return nil, err
		}

		if len(results) == 0 {
			continue
		}

		// 构建参赛者列表
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

		// 计算 Rating 变化
		changes, err := e.calculator.Calculate(ctx, &contest, participants, e.config)
		if err != nil {
			return nil, err
		}

		// 更新内存中的学生状态
		for _, c := range changes {
			student := studentRatings[c.StudentID]
			if student != nil {
				if c.RatingAfter > student.CurrentRating {
					student.CurrentRating = c.RatingAfter
					if c.RatingAfter > student.MaxRating {
						student.MaxRating = c.RatingAfter
					}
				}
				student.MatchCount++
			}
		}
	}

	// 转换为切片并按 Rating 排序
	var result []PreviewStudent
	for _, s := range studentRatings {
		if s.MatchCount > 0 {
			result = append(result, *s)
		}
	}

	// 按 Rating 降序排序
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].CurrentRating > result[i].CurrentRating {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result, nil
}
