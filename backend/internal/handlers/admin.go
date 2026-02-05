package handlers

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/time/rate"

	"rating-system/internal/config"
	"rating-system/internal/database"
	"rating-system/internal/models"
	"rating-system/internal/services/csv"
	"rating-system/internal/services/rating"
)

var adminConfig *config.AdminConfig
var loginLimiter = newIPRateLimiter(5, 10)

// SetAdminConfig 设置管理员配置
func SetAdminConfig(cfg *config.AdminConfig) {
	adminConfig = cfg
}

// AdminLogin 管理员登录
// @Summary 管理员登录
// @Tags Admin
// @Param body body map[string]string true "登录信息"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/login [post]
func AdminLogin(c *gin.Context) {
	if !loginLimiter.Allow(clientIP(c)) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "登录过于频繁，请稍后再试"})
		return
	}

	if !isBcryptHash(adminConfig.Password) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "管理员密码未加密，请设置 bcrypt 哈希"})
		return
	}

	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	if !verifyAdminPassword(req.Password, adminConfig.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
		return
	}

	expiry := time.Duration(adminConfig.TokenExp) * time.Hour
	claims := jwt.RegisteredClaims{
		Subject:   "admin",
		Issuer:    "rating-system",
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(adminConfig.JWTSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成令牌失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      tokenStr,
		"expires_in": adminConfig.TokenExp * 3600,
	})
}

// GetConfig 获取算法配置
// @Summary 获取算法配置
// @Tags Admin
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/config [get]
func GetConfig(c *gin.Context) {
	db := database.GetDB()
	var configs []models.SystemConfig
	db.Find(&configs)

	result := make(map[string]string)
	for _, cfg := range configs {
		result[cfg.Key] = cfg.Value
	}

	c.JSON(http.StatusOK, result)
}

// UpdateConfig 更新算法配置
// @Summary 更新算法配置
// @Tags Admin
// @Param body body map[string]interface{} true "配置信息"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/config [put]
func UpdateConfig(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	db := database.GetDB()
	for key, value := range req {
		var strValue string
		switch v := value.(type) {
		case float64:
			strValue = strconv.FormatFloat(v, 'f', -1, 64)
		case string:
			strValue = v
		default:
			continue
		}

		db.Model(&models.SystemConfig{}).Where("key = ?", key).Update("value", strValue)
	}

	c.JSON(http.StatusOK, gin.H{"message": "配置更新成功，请执行重算以应用新参数"})
}

// UpdateContest 更新比赛信息
// @Summary 更新比赛信息
// @Tags Admin
// @Param id path int true "比赛ID"
// @Param body body map[string]interface{} true "比赛信息"
// @Success 200 {object} models.Contest
// @Router /api/v1/admin/contests/{id} [put]
func UpdateContest(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Name   string  `json:"name"`
		Weight float64 `json:"weight"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	db := database.GetDB()
	var contest models.Contest
	if err := db.First(&contest, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "比赛不存在"})
		return
	}

	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Weight > 0 {
		updates["weight"] = req.Weight
	}

	db.Model(&contest).Updates(updates)

	c.JSON(http.StatusOK, gin.H{
		"contest": contest,
		"message": "比赛更新成功，如修改了权重请执行重算",
	})
}

// DeleteContest 删除比赛
// @Summary 删除比赛
// @Tags Admin
// @Param id path int true "比赛ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/contests/{id} [delete]
func DeleteContest(c *gin.Context) {
	id := c.Param("id")

	db := database.GetDB()

	// 删除相关结果
	db.Where("contest_id = ?", id).Delete(&models.Result{})

	// 删除比赛
	if err := db.Delete(&models.Contest{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "比赛删除成功，请执行重算以更新 Rating"})
}

// UpdateTiers 更新段位配置
// @Summary 更新段位配置
// @Tags Admin
// @Param body body []models.TierConfig true "段位配置"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/tiers [put]
func UpdateTiers(c *gin.Context) {
	var tiers []models.TierConfig
	if err := c.ShouldBindJSON(&tiers); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	db := database.GetDB()

	// 清空现有段位
	db.Exec("DELETE FROM tier_configs")

	// 插入新段位
	for _, tier := range tiers {
		db.Create(&tier)
	}

	c.JSON(http.StatusOK, gin.H{"message": "段位配置更新成功"})
}

// UploadContest 上传比赛 CSV
// @Summary 上传比赛 CSV
// @Tags Admin
// @Accept multipart/form-data
// @Param file formData file true "CSV 文件"
// @Param name formData string true "比赛名称"
// @Param date formData string true "比赛日期 (YYYY-MM-DD)"
// @Param weight formData number false "比赛权重" default(1.0)
// @Success 200 {object} models.Contest
// @Router /api/v1/admin/contests [post]
func UploadContest(c *gin.Context) {
	// 获取文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传 CSV 文件"})
		return
	}
	defer file.Close()

	// 获取参数
	name := c.PostForm("name")
	if name == "" {
		name = header.Filename
	}

	dateStr := c.PostForm("date")
	date := time.Now()
	if dateStr != "" {
		if parsed, err := time.Parse("2006-01-02", dateStr); err == nil {
			date = parsed
		}
	}

	weight := 1.0
	if w := c.PostForm("weight"); w != "" {
		if parsed, err := strconv.ParseFloat(w, 64); err == nil {
			weight = parsed
		}
	}

	// 保存文件
	uploadDir := "../data/uploads"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建上传目录失败"})
		return
	}

	filename := filepath.Join(uploadDir, time.Now().Format("20060102_150405")+"_"+header.Filename)
	dst, err := os.Create(filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败"})
		return
	}
	defer dst.Close()

	// 重置文件指针
	file.Seek(0, 0)
	if _, err := io.Copy(dst, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败"})
		return
	}

	// 解析 CSV
	file.Seek(0, 0)
	parser := csv.GetParser("hydro")
	if parser == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "CSV 解析器未找到"})
		return
	}

	entries, err := parser.Parse(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CSV 解析失败: " + err.Error()})
		return
	}

	if len(entries) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CSV 文件无有效数据（所有人都是0分？）"})
		return
	}

	db := database.GetDB()

	// 创建比赛
	contest := models.Contest{
		Name:             name,
		Date:             date,
		Weight:           weight,
		CSVPath:          filename,
		ParticipantCount: len(entries),
	}
	if err := db.Create(&contest).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建比赛失败"})
		return
	}

	// 创建/更新学生和结果
	for _, entry := range entries {
		// 创建或更新学生
		student := models.Student{
			StudentID: entry.StudentID,
			Name:      entry.Name,
			Email:     entry.Email,
			Class:     entry.Class,
			Grade:     entry.Grade,
		}
		db.Where("student_id = ?", entry.StudentID).FirstOrCreate(&student)

		// 更新班级信息（如果已存在但班级为空）
		if student.Class == "" && entry.Class != "" {
			db.Model(&student).Update("class", entry.Class)
		}

		// 创建结果（Rating 相关字段在重算时填充）
		result := models.Result{
			StudentID: entry.StudentID,
			ContestID: contest.ID,
			Rank:      entry.Rank,
			Solved:    entry.Solved,
			TotalTime: entry.TotalTime,
		}
		db.Create(&result)
	}

	c.JSON(http.StatusOK, gin.H{
		"contest":      contest,
		"participants": len(entries),
		"message":      "比赛创建成功，请执行重算以计算 Rating",
	})
}

// replayEngine 全局重算引擎
var replayEngine *rating.ReplayEngine

// TriggerReplay 触发重算
// @Summary 触发全量重算
// @Tags Admin
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/replay/apply [post]
func TriggerReplay(c *gin.Context) {
	db := database.GetDB()

	if replayEngine == nil {
		replayEngine = rating.NewReplayEngine(db, rating.NewEloMMRCalculator())
	}

	if replayEngine.IsRunning() {
		c.JSON(http.StatusConflict, gin.H{"error": "重算正在进行中，请稍后再试"})
		return
	}

	// 异步执行重算
	go func() {
		progressCh := make(chan rating.ReplayProgress, 10)

		go func() {
			for range progressCh {
				// 可以在这里添加 SSE 推送逻辑
			}
		}()

		if err := replayEngine.ReplayAll(context.Background(), progressCh); err != nil {
			// 记录错误日志
		}
		close(progressCh)
	}()

	c.JSON(http.StatusOK, gin.H{"message": "重算已启动"})
}

// GetReplayStatus 获取重算状态
// @Summary 获取重算状态
// @Tags Admin
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/replay/status [get]
func GetReplayStatus(c *gin.Context) {
	if replayEngine == nil {
		c.JSON(http.StatusOK, gin.H{"running": false, "progress": rating.ReplayProgress{Status: "idle"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"running":  replayEngine.IsRunning(),
		"progress": replayEngine.Progress(),
	})
}

// PreviewReplay 预览重算结果（不实际保存）
// @Summary 预览重算结果
// @Tags Admin
// @Param body body map[string]interface{} true "预览配置"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/replay/preview [post]
func PreviewReplay(c *gin.Context) {
	var req struct {
		KFactor            float64 `json:"k_factor"`
		GrowthInertia      float64 `json:"growth_inertia"`
		HistoryDecay       float64 `json:"history_decay"`
		InitialRating      float64 `json:"initial_rating"`
		TopBonusThreshold  float64 `json:"top_bonus_threshold"`
		TopBonusMultiplier float64 `json:"top_bonus_multiplier"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	db := database.GetDB()

	// 使用临时配置创建预览引擎
	previewEngine := rating.NewReplayEngine(db, rating.NewEloMMRCalculator())
	previewEngine.SetConfig(rating.RatingConfig{
		KFactor:            req.KFactor,
		GrowthInertia:      req.GrowthInertia,
		HistoryDecay:       req.HistoryDecay,
		InitialRating:      req.InitialRating,
		TopBonusThreshold:  req.TopBonusThreshold,
		TopBonusMultiplier: req.TopBonusMultiplier,
	})

	// 执行预览计算（不保存到数据库）
	results, err := previewEngine.PreviewAll(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "预览计算失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"preview": results,
		"config": gin.H{
			"k_factor":             req.KFactor,
			"growth_inertia":       req.GrowthInertia,
			"history_decay":        req.HistoryDecay,
			"initial_rating":       req.InitialRating,
			"top_bonus_threshold":  req.TopBonusThreshold,
			"top_bonus_multiplier": req.TopBonusMultiplier,
		},
	})
}

func verifyAdminPassword(plain, stored string) bool {
	stored = strings.TrimSpace(stored)
	if strings.HasPrefix(stored, "bcrypt:") {
		hash := strings.TrimPrefix(stored, "bcrypt:")
		return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
	}
	if strings.HasPrefix(stored, "$2a$") || strings.HasPrefix(stored, "$2b$") || strings.HasPrefix(stored, "$2y$") {
		return bcrypt.CompareHashAndPassword([]byte(stored), []byte(plain)) == nil
	}
	return false
}

func isBcryptHash(stored string) bool {
	stored = strings.TrimSpace(stored)
	return strings.HasPrefix(stored, "bcrypt:") ||
		strings.HasPrefix(stored, "$2a$") ||
		strings.HasPrefix(stored, "$2b$") ||
		strings.HasPrefix(stored, "$2y$")
}

type ipRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	rate     rate.Limit
	burst    int
}

func newIPRateLimiter(r rate.Limit, b int) *ipRateLimiter {
	return &ipRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     r,
		burst:    b,
	}
}

func (l *ipRateLimiter) getLimiter(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()
	limiter, ok := l.limiters[ip]
	if !ok {
		limiter = rate.NewLimiter(l.rate, l.burst)
		l.limiters[ip] = limiter
	}
	return limiter
}

func (l *ipRateLimiter) Allow(ip string) bool {
	return l.getLimiter(ip).Allow()
}

func clientIP(c *gin.Context) string {
	ip := c.ClientIP()
	if ip == "" {
		ip = c.Request.RemoteAddr
	}
	return ip
}

// UpdateSiteConfig 更新网站配置
// @Summary 更新网站配置
// @Tags Admin
// @Param body body map[string]interface{} true "网站配置"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/site-config [put]
func UpdateSiteConfig(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	db := database.GetDB()
	for key, value := range req {
		var strValue string
		switch v := value.(type) {
		case bool:
			strValue = strconv.FormatBool(v)
		case float64:
			strValue = strconv.FormatFloat(v, 'f', -1, 64)
		case string:
			strValue = v
		default:
			continue
		}

		// 添加 site_ 前缀
		dbKey := "site_" + key
		// 使用 upsert
		var existing models.SystemConfig
		if err := db.Where("key = ?", dbKey).First(&existing).Error; err != nil {
			// 不存在，创建
			db.Create(&models.SystemConfig{Key: dbKey, Value: strValue})
		} else {
			// 存在，更新
			db.Model(&existing).Update("value", strValue)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "网站配置更新成功"})
}
