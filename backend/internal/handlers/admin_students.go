package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"rating-system/internal/database"
	"rating-system/internal/models"
	"rating-system/internal/services/points"
)

func AdminListStudents(c *gin.Context) {
	page := 1
	size := 20
	if p := c.Query("page"); p != "" {
		if pInt, err := parseIntParamAdmin(p); err == nil && pInt > 0 {
			page = pInt
		}
	}
	if s := c.Query("size"); s != "" {
		if sInt, err := parseIntParamAdmin(s); err == nil && sInt > 0 && sInt <= 200 {
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
	query.Order("updated_at DESC").Offset((page - 1) * size).Limit(size).Find(&students)

	c.JSON(http.StatusOK, gin.H{
		"total": total,
		"page":  page,
		"size":  size,
		"data":  students,
	})
}

func AdminCreateStudent(c *gin.Context) {
	var req struct {
		StudentID string `json:"student_id" binding:"required"`
		Name      string `json:"name" binding:"required"`
		Email     string `json:"email"`
		Class     string `json:"class"`
		Grade     *int   `json:"grade"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	db := database.GetDB()
	var existing models.Student
	if err := db.First(&existing, "student_id = ?", req.StudentID).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "学号已存在"})
		return
	}

	grade := 0
	if req.Grade != nil {
		grade = *req.Grade
	} else {
		grade = models.ParseGrade(req.StudentID)
	}

	student := models.Student{
		StudentID: req.StudentID,
		Name:      req.Name,
		Email:     req.Email,
		Class:     req.Class,
		Grade:     grade,
	}
	if err := db.Create(&student).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建学生失败"})
		return
	}

	c.JSON(http.StatusOK, student)
}

func AdminUpdateStudent(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Name  *string `json:"name"`
		Email *string `json:"email"`
		Class *string `json:"class"`
		Grade *int    `json:"grade"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Email != nil {
		updates["email"] = *req.Email
	}
	if req.Class != nil {
		updates["class"] = *req.Class
	}
	if req.Grade != nil {
		updates["grade"] = *req.Grade
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有可更新字段"})
		return
	}

	db := database.GetDB()
	if err := db.Model(&models.Student{}).Where("student_id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新学生失败"})
		return
	}

	var student models.Student
	if err := db.First(&student, "student_id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "学生不存在"})
		return
	}

	c.JSON(http.StatusOK, student)
}

func AdminCreateStudentPointRecord(c *gin.Context) {
	studentID := c.Param("id")

	var req struct {
		Category    models.PointCategory `json:"category" binding:"required"`
		Points      int                  `json:"points" binding:"required"`
		Title       string               `json:"title" binding:"required"`
		Description string               `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}
	if req.Points <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "积分必须大于 0"})
		return
	}
	if req.Category != models.PointCategoryProgressAward && req.Category != models.PointCategoryOrganizerReward {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持手工录入进步奖或组织工作积分"})
		return
	}

	db := database.GetDB()
	var student models.Student
	if err := db.First(&student, "student_id = ?", studentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "学生不存在"})
		return
	}

	record := models.PointRecord{
		StudentID:   studentID,
		Category:    req.Category,
		Points:      req.Points,
		Title:       req.Title,
		Description: req.Description,
	}
	if err := points.NewService(db).CreateManualRecord(&record); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "积分录入失败"})
		return
	}
	if err := db.First(&student, "student_id = ?", studentID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "积分录入失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"record":  record,
		"student": student,
	})
}

func AdminDeleteStudent(c *gin.Context) {
	id := c.Param("id")

	db := database.GetDB()
	db.Where("student_id = ?", id).Delete(&models.PointRecord{})
	db.Where("student_id = ?", id).Delete(&models.Result{})
	if err := db.Delete(&models.Student{}, "student_id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除学生失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func AdminDeleteAllStudents(c *gin.Context) {
	db := database.GetDB()
	db.Exec("DELETE FROM point_records")
	db.Exec("DELETE FROM results")
	if err := db.Exec("DELETE FROM students").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除学生失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已删除全部学生"})
}

func parseIntParamAdmin(s string) (int, error) {
	return strconv.Atoi(s)
}
