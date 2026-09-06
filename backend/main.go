package main

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"

	"leans/config"
	"leans/handler"
	"leans/model"
	"leans/service"
	"leans/storage"
	"leans/subject"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	store, err := storage.NewStore(cfg.Data.Dir)
	if err != nil {
		log.Fatalf("init store: %v", err)
	}
	defer store.Close()

	// NewStore 容忍目录缺失（返回空 store）；仅读取失败时降级为空 store，
	// 保证应用仍能启动，讲义列表为空。
	subjects, err := subject.NewStore(cfg.Subjects.Dir)
	if err != nil {
		log.Printf("load subjects warning: %v, use empty store", err)
		subjects, _ = subject.NewStore("")
	}
	list := subjects.List()
	log.Printf("Loaded %d subjects", len(list))

	// config.yaml 的 AI 配置作为种子供应商；用户在界面保存设置后被
	// 持久化的多供应商配置完全取代。
	var defaults model.AISettings
	if cfg.AI.BaseURL != "" {
		defaults.Providers = []model.AIProvider{{
			ID:      "default",
			Name:    cfg.AI.Provider,
			BaseURL: cfg.AI.BaseURL,
			APIKey:  cfg.AI.APIKey,
			Models:  []string{cfg.AI.Model},
		}}
		defaults.ActiveProviderID = "default"
		defaults.ActiveModel = cfg.AI.Model
	}

	ai := service.NewAIService(defaults, store.GetSettings, cfg.Analysis.MaxTokens)
	analyzer := service.NewAnalyzer(ai, subjects, store, cfg.Analysis)

	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	api := r.Group("/api")
	{
		subjectHandler := handler.NewSubjectHandler(subjects)
		api.GET("/subjects", subjectHandler.List)
		api.GET("/subjects/:id", subjectHandler.GetContent)

		analyzeHandler := handler.NewAnalyzeHandler(analyzer)
		api.POST("/analyze", analyzeHandler.Handle)
		api.POST("/analyze/stream", analyzeHandler.HandleStream)

		settingsHandler := handler.NewSettingsHandler(store, ai)
		api.GET("/settings", settingsHandler.Get)
		api.PUT("/settings", settingsHandler.Put)
		api.POST("/settings/active", settingsHandler.Active)
		api.POST("/settings/models", settingsHandler.Models)
		api.POST("/settings/test", settingsHandler.Test)

		historyHandler := handler.NewHistoryHandler(store)
		api.GET("/history", historyHandler.List)
		api.GET("/history/:id", historyHandler.Get)
		api.DELETE("/history", historyHandler.Clear)

		statsHandler := handler.NewStatsHandler(store)
		api.GET("/stats", statsHandler.Get)
	}

	staticFS, err := fs.Sub(StaticFS, "static")
	if err != nil {
		log.Fatalf("load static: %v", err)
	}

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if f, err := staticFS.(fs.ReadFileFS).ReadFile(path[1:]); err == nil {
			c.Data(http.StatusOK, guessContentType(path), f)
			return
		}
		indexHTML, _ := staticFS.(fs.ReadFileFS).ReadFile("index.html")
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	})

	r.GET("/", func(c *gin.Context) {
		indexHTML, _ := staticFS.(fs.ReadFileFS).ReadFile("index.html")
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	})

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Server starting on http://localhost:%d", cfg.Server.Port)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func guessContentType(path string) string {
	ext := filepath.Ext(path)
	switch ext {
	case ".html":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".svg":
		return "image/svg+xml"
	case ".ico":
		return "image/x-icon"
	default:
		return "application/octet-stream"
	}
}
