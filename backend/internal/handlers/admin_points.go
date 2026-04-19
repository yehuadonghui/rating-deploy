package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"rating-system/internal/database"
	"rating-system/internal/models"
	"rating-system/internal/services/points"
)

func AdminListPointRecords(c *gin.Context) {
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

	query := points.ListQuery{
		StudentID: c.Query("student_id"),
		Category:  models.PointCategory(c.Query("category")),
		Source:    models.PointSource(c.Query("source")),
		Page:      page,
		Size:      size,
	}
	if contestID := c.Query("contest_id"); contestID != "" {
		query.ContestID = parseUint(contestID)
	}

	service := points.NewService(database.GetDB())
	records, total, err := service.ListPointRecords(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取积分流水失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total": total,
		"page":  page,
		"size":  size,
		"data":  records,
	})
}
