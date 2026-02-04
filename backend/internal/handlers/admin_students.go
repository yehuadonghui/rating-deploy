package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"rating-system/internal/database"
	"rating-system/internal/models"
)

// AdminListStudents 管理端获取学生列表
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
	query.Order("updated_at DESC").
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

// AdminCreateStudent 管理端新增学生
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

// AdminUpdateStudent 管理端更新学生
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

// AdminDeleteStudent 管理端删除学生
func AdminDeleteStudent(c *gin.Context) {
	id := c.Param("id")

	db := database.GetDB()
	db.Where("student_id = ?", id).Delete(&models.Result{})
	if err := db.Delete(&models.Student{}, "student_id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除学生失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func parseIntParamAdmin(s string) (int, error) {
	return strconv.Atoi(s)
}
