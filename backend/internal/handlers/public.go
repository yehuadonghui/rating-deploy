package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"rating-system/internal/database"
	"rating-system/internal/models"
	"rating-system/internal/services/points"
)

// GetLeaderboard 获取排行榜
// @Summary 获取排行榜
// @Tags Public
// @Param grade query int false "年级筛选"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(20)
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/leaderboard [get]
func GetLeaderboard(c *gin.Context) {
	var grade *int
	if g := c.Query("grade"); g != "" {
		if gInt, err := strconv.Atoi(g); err == nil {
			grade = &gInt
		}
	}

	page := 1
	size := 20
	if p := c.Query("page"); p != "" {
		if pInt, err := parseIntParam(p); err == nil && pInt > 0 {
			page = pInt
		}
	}
	if s := c.Query("size"); s != "" {
		if sInt, err := parseIntParam(s); err == nil && sInt > 0 && sInt <= 100 {
			size = sInt
		}
	}

	db := database.GetDB()
	query := db.Model(&models.Student{}).Where("match_count > 0")

	if grade != nil {
		query = query.Where("grade = ?", *grade)
	}

	var total int64
	query.Count(&total)

	var students []models.Student
	// 主排序：current_rating DESC，次排序：最近表现（通过子查询）
	query.Order("current_rating DESC, updated_at DESC").
		Offset((page - 1) * size).
		Limit(size).
		Find(&students)

	// 获取段位配置
	var tiers []models.TierConfig
	db.Order("`order` ASC").Find(&tiers)

	// 为每个学生添加段位信息
	type StudentWithTier struct {
		models.Student
		TierName  string `json:"tier_name"`
		TierColor string `json:"tier_color"`
		Rank      int    `json:"rank"`
	}

	var result []StudentWithTier
	for i, s := range students {
		swt := StudentWithTier{
			Student: s,
			Rank:    (page-1)*size + i + 1,
		}
		// 匹配段位
		for _, t := range tiers {
			if int(s.CurrentRating) >= t.MinRating && int(s.CurrentRating) <= t.MaxRating {
				swt.TierName = t.Name
				swt.TierColor = t.Color
				break
			}
		}
		result = append(result, swt)
	}

	c.JSON(http.StatusOK, gin.H{
		"total": total,
		"page":  page,
		"size":  size,
		"data":  result,
	})
}

// GetStudent 获取学生详情
// @Summary 获取学生详情
// @Tags Public
// @Param id path string true "学号"
// @Success 200 {object} models.Student
// @Router /api/v1/students/{id} [get]
func GetStudent(c *gin.Context) {
	id := c.Param("id")

	db := database.GetDB()
	var student models.Student
	if err := db.First(&student, "student_id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "学生不存在"})
		return
	}

	// 获取段位
	var tiers []models.TierConfig
	db.Order("`order` ASC").Find(&tiers)

	tierName := ""
	tierColor := ""
	for _, t := range tiers {
		if int(student.CurrentRating) >= t.MinRating && int(student.CurrentRating) <= t.MaxRating {
			tierName = t.Name
			tierColor = t.Color
			break
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"student":    student,
		"tier_name":  tierName,
		"tier_color": tierColor,
	})
}

// GetStudentHistory 获取学生 Rating 历史
// @Summary 获取学生 Rating 历史
// @Tags Public
// @Param id path string true "学号"
// @Success 200 {array} models.Result
// @Router /api/v1/students/{id}/history [get]
func GetStudentHistory(c *gin.Context) {
	id := c.Param("id")

	db := database.GetDB()
	var results []models.Result
	db.Preload("Contest").
		Where("student_id = ?", id).
		Order("created_at ASC").
		Find(&results)

	c.JSON(http.StatusOK, results)
}

func GetStudentPointRecords(c *gin.Context) {
	id := c.Param("id")

	db := database.GetDB()
	var student models.Student
	if err := db.First(&student, "student_id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "学生不存在"})
		return
	}

	records, err := points.NewService(db).ListStudentPointRecords(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取兑奖积分明细失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"student_id":    id,
		"redeem_points": student.RedeemPoints,
		"point_records": records,
	})
}

// GetContests 获取比赛列表
// @Summary 获取比赛列表
// @Tags Public
// @Success 200 {array} models.Contest
// @Router /api/v1/contests [get]
func GetContests(c *gin.Context) {
	db := database.GetDB()
	var contests []models.Contest
	db.Order("date DESC").Find(&contests)
	c.JSON(http.StatusOK, contests)
}

// GetTiers 获取段位配置
// @Summary 获取段位配置
// @Tags Public
// @Success 200 {array} models.TierConfig
// @Router /api/v1/tiers [get]
func GetTiers(c *gin.Context) {
	db := database.GetDB()
	var tiers []models.TierConfig
	db.Order("`order` ASC").Find(&tiers)
	c.JSON(http.StatusOK, tiers)
}

// GetStatistics 获取统计数据
// @Summary 获取统计数据
// @Tags Public
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/statistics [get]
func GetStatistics(c *gin.Context) {
	db := database.GetDB()

	// 总学生数
	var totalStudents int64
	db.Model(&models.Student{}).Where("match_count > 0").Count(&totalStudents)

	// 总比赛数
	var totalContests int64
	db.Model(&models.Contest{}).Count(&totalContests)

	// 各年级分布
	type GradeCount struct {
		Grade     int     `json:"grade"`
		Count     int64   `json:"count"`
		AvgRating float64 `json:"avg_rating"`
	}
	var gradeDistribution []GradeCount
	db.Model(&models.Student{}).
		Select("grade, COUNT(*) as count, AVG(current_rating) as avg_rating").
		Where("match_count > 0").
		Group("grade").
		Order("grade DESC").
		Scan(&gradeDistribution)

	// 段位分布
	var tiers []models.TierConfig
	db.Order("`order` ASC").Find(&tiers)

	type TierCount struct {
		Tier  string `json:"tier"`
		Color string `json:"color"`
		Count int64  `json:"count"`
	}
	var tierDistribution []TierCount
	for _, t := range tiers {
		var count int64
		db.Model(&models.Student{}).
			Where("match_count > 0 AND current_rating >= ? AND current_rating <= ?", t.MinRating, t.MaxRating).
			Count(&count)
		tierDistribution = append(tierDistribution, TierCount{
			Tier:  t.Name,
			Color: t.Color,
			Count: count,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"total_students":     totalStudents,
		"total_contests":     totalContests,
		"grade_distribution": gradeDistribution,
		"tier_distribution":  tierDistribution,
	})
}

// GetSiteConfig 获取网站配置
// @Summary 获取网站配置
// @Tags Public
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/site-config [get]
func GetSiteConfig(c *gin.Context) {
	db := database.GetDB()
	var configs []models.SystemConfig
	db.Where("key LIKE ?", "site_%").Find(&configs)

	result := make(map[string]string)
	for _, cfg := range configs {
		// 去掉 site_ 前缀
		key := cfg.Key[5:]
		result[key] = cfg.Value
	}

	c.JSON(http.StatusOK, result)
}

// SearchStudents 学生搜索
// @Summary 学生搜索
// @Tags Public
// @Param query query string false "关键词"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(20)
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/students [get]
func SearchStudents(c *gin.Context) {
	page := 1
	size := 20
	if p := c.Query("page"); p != "" {
		if pInt, err := parseIntParam(p); err == nil && pInt > 0 {
			page = pInt
		}
	}
	if s := c.Query("size"); s != "" {
		if sInt, err := parseIntParam(s); err == nil && sInt > 0 && sInt <= 100 {
			size = sInt
		}
	}

	queryStr := c.Query("query")

	db := database.GetDB()
	query := db.Model(&models.Student{})

	if queryStr != "" {
		like := "%" + queryStr + "%"
		query = query.Where(
			"student_id LIKE ? OR name LIKE ? OR class LIKE ? OR email LIKE ?",
			like, like, like, like,
		)
	}

	var total int64
	query.Count(&total)

	var students []models.Student
	query.Order("current_rating DESC").
		Offset((page - 1) * size).
		Limit(size).
		Find(&students)

	c.JSON(http.StatusOK, gin.H{
		"total": total,
		"page":  page,
		"size":  size,
		"data":  students,
	})
}

func parseIntParam(s string) (int, error) {
	return strconv.Atoi(s)
}
