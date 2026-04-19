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
	"rating-system/internal/services/points"
	"rating-system/internal/services/rating"
)

var adminConfig *config.AdminConfig
var loginLimiter = newIPRateLimiter(5, 10)

func SetAdminConfig(cfg *config.AdminConfig) {
	adminConfig = cfg
}

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

func GetConfig(c *gin.Context) {
	db := database.GetDB()
	var configs []models.SystemConfig
	db.Find(&configs)

	result := make(map[string]string, len(configs))
	for _, cfg := range configs {
		result[cfg.Key] = cfg.Value
	}
	c.JSON(http.StatusOK, result)
}

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

func UpdateContest(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Name       string                   `json:"name"`
		Weight     float64                  `json:"weight"`
		RewardType models.ContestRewardType `json:"reward_type"`
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
	if req.RewardType != "" {
		updates["reward_type"] = req.RewardType
	}

	if err := db.Model(&contest).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "比赛更新失败"})
		return
	}
	if err := db.First(&contest, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "比赛更新失败"})
		return
	}
	if err := points.NewService(db).ReplaceContestAutoRecords(&contest); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "兑奖积分更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"contest": contest,
		"message": "比赛更新成功，如修改了权重请执行重算",
	})
}

func DeleteContest(c *gin.Context) {
	id := c.Param("id")
	contestID := parseUint(id)

	db := database.GetDB()
	if err := points.NewService(db).DeleteContestAutoRecords(contestID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "兑奖积分删除失败"})
		return
	}
	db.Where("contest_id = ?", id).Delete(&models.Result{})
	if err := db.Delete(&models.Contest{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "比赛删除成功，请执行重算以更新 Rating"})
}

func UpdateTiers(c *gin.Context) {
	var tiers []models.TierConfig
	if err := c.ShouldBindJSON(&tiers); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	db := database.GetDB()
	db.Exec("DELETE FROM tier_configs")
	for _, tier := range tiers {
		db.Create(&tier)
	}

	c.JSON(http.StatusOK, gin.H{"message": "段位配置更新成功"})
}

func UploadContest(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传 CSV 文件"})
		return
	}
	defer file.Close()

	name := c.PostForm("name")
	if name == "" {
		name = header.Filename
	}

	date := time.Now()
	if dateStr := c.PostForm("date"); dateStr != "" {
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

	rewardType := models.ContestRewardType(c.PostForm("reward_type"))
	if rewardType == "" {
		rewardType = models.ContestRewardTypeNone
	}

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

	if _, err := file.Seek(0, 0); err == nil {
		if _, err := io.Copy(dst, file); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败"})
			return
		}
	}

	if _, err := file.Seek(0, 0); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件失败"})
		return
	}

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
		c.JSON(http.StatusBadRequest, gin.H{"error": "CSV 文件无有效数据"})
		return
	}

	db := database.GetDB()
	contest := models.Contest{
		Name:             name,
		Date:             date,
		Weight:           weight,
		RewardType:       rewardType,
		CSVPath:          filename,
		ParticipantCount: len(entries),
	}
	if err := db.Create(&contest).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建比赛失败"})
		return
	}

	for _, entry := range entries {
		student := models.Student{
			StudentID: entry.StudentID,
			Name:      entry.Name,
			Email:     entry.Email,
			Class:     entry.Class,
			Grade:     entry.Grade,
		}
		if err := db.Where("student_id = ?", entry.StudentID).FirstOrCreate(&student).Error; err != nil {
			continue
		}

		updates := make(map[string]interface{})
		if entry.Name != "" {
			updates["name"] = entry.Name
		}
		if entry.Email != "" {
			updates["email"] = entry.Email
		}
		if entry.Class != "" {
			updates["class"] = entry.Class
		}
		if entry.Grade != 0 {
			updates["grade"] = entry.Grade
		}
		if len(updates) > 0 {
			db.Model(&models.Student{}).Where("student_id = ?", entry.StudentID).Updates(updates)
		}

		result := models.Result{
			StudentID: entry.StudentID,
			ContestID: contest.ID,
			Rank:      entry.Rank,
			Solved:    entry.Solved,
			TotalTime: entry.TotalTime,
		}
		db.Create(&result)
	}

	if err := points.NewService(db).ReplaceContestAutoRecords(&contest); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "兑奖积分计算失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"contest":      contest,
		"participants": len(entries),
		"message":      "比赛创建成功，请执行重算以计算 Rating",
	})
}

var replayEngine *rating.ReplayEngine

func TriggerReplay(c *gin.Context) {
	db := database.GetDB()
	if replayEngine == nil {
		replayEngine = rating.NewReplayEngine(db, rating.NewEloMMRCalculator())
	}
	if replayEngine.IsRunning() {
		c.JSON(http.StatusConflict, gin.H{"error": "重算正在进行中，请稍后再试"})
		return
	}

	go func() {
		progressCh := make(chan rating.ReplayProgress, 10)
		go func() {
			for range progressCh {
			}
		}()
		_ = replayEngine.ReplayAll(context.Background(), progressCh)
		close(progressCh)
	}()

	c.JSON(http.StatusOK, gin.H{"message": "重算已启动"})
}

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
	previewEngine := rating.NewReplayEngine(db, rating.NewEloMMRCalculator())
	previewEngine.SetConfig(rating.RatingConfig{
		KFactor:            req.KFactor,
		GrowthInertia:      req.GrowthInertia,
		HistoryDecay:       req.HistoryDecay,
		InitialRating:      req.InitialRating,
		TopBonusThreshold:  req.TopBonusThreshold,
		TopBonusMultiplier: req.TopBonusMultiplier,
	})

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

		dbKey := "site_" + key
		var existing models.SystemConfig
		if err := db.Where("key = ?", dbKey).First(&existing).Error; err != nil {
			db.Create(&models.SystemConfig{Key: dbKey, Value: strValue})
		} else {
			db.Model(&existing).Update("value", strValue)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "网站配置更新成功"})
}

func parseUint(value string) uint {
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0
	}
	return uint(parsed)
}
