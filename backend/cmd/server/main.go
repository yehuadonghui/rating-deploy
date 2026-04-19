package main

import (
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"rating-system/internal/config"
	"rating-system/internal/database"
	"rating-system/internal/handlers"
)

func main() {
	cfg := config.Load()

	gin.SetMode(cfg.Server.Mode)

	if err := database.Init(cfg); err != nil {
		log.Fatalf("database init failed: %v", err)
	}
	if err := database.Migrate(); err != nil {
		log.Fatalf("database migrate failed: %v", err)
	}

	handlers.SetAdminConfig(&cfg.Admin)

	router := gin.Default()
	router.Use(handlers.CORSMiddleware())

	v1 := router.Group("/api/v1")
	v1.GET("/leaderboard", handlers.GetLeaderboard)
	v1.GET("/students/:id", handlers.GetStudent)
	v1.GET("/students/:id/history", handlers.GetStudentHistory)
	v1.GET("/students/:id/points", handlers.GetStudentPointRecords)
	v1.GET("/contests", handlers.GetContests)
	v1.GET("/tiers", handlers.GetTiers)
	v1.GET("/statistics", handlers.GetStatistics)
	v1.GET("/site-config", handlers.GetSiteConfig)
	v1.GET("/students", handlers.SearchStudents)

	admin := v1.Group("/admin")
	admin.POST("/login", handlers.AdminLogin)

	adminAuth := admin.Group("")
	adminAuth.Use(handlers.AuthMiddleware(&cfg.Admin))
	adminAuth.GET("/config", handlers.GetConfig)
	adminAuth.PUT("/config", handlers.UpdateConfig)
	adminAuth.POST("/contests", handlers.UploadContest)
	adminAuth.PUT("/contests/:id", handlers.UpdateContest)
	adminAuth.DELETE("/contests/:id", handlers.DeleteContest)
	adminAuth.PUT("/tiers", handlers.UpdateTiers)
	adminAuth.POST("/replay/apply", handlers.TriggerReplay)
	adminAuth.POST("/replay/preview", handlers.PreviewReplay)
	adminAuth.GET("/replay/status", handlers.GetReplayStatus)
	adminAuth.PUT("/site-config", handlers.UpdateSiteConfig)
	adminAuth.GET("/students", handlers.AdminListStudents)
	adminAuth.GET("/points", handlers.AdminListPointRecords)
	adminAuth.POST("/students", handlers.AdminCreateStudent)
	adminAuth.PUT("/students/:id", handlers.AdminUpdateStudent)
	adminAuth.POST("/students/:id/points", handlers.AdminCreateStudentPointRecord)
	adminAuth.DELETE("/students/:id", handlers.AdminDeleteStudent)
	adminAuth.DELETE("/students", handlers.AdminDeleteAllStudents)

	distDir := filepath.Clean("../frontend/dist")
	indexPath := filepath.Join(distDir, "index.html")
	router.Static("/assets", filepath.Join(distDir, "assets"))
	router.StaticFile("/vite.svg", filepath.Join(distDir, "vite.svg"))
	router.GET("/", func(c *gin.Context) {
		c.File(indexPath)
	})
	router.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		c.File(indexPath)
	})

	addr := ":" + cfg.Server.Port
	log.Printf("server listening on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
