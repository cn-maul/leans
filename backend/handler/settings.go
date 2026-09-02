package handler

import (
	"net/http"

	"leans/model"
	"leans/service"
	"leans/storage"

	"github.com/gin-gonic/gin"
)

type SettingsHandler struct {
	store *storage.Store
	ai    *service.AIService
}

func NewSettingsHandler(store *storage.Store, ai *service.AIService) *SettingsHandler {
	return &SettingsHandler{store: store, ai: ai}
}

// Get returns the merged settings (defaults + persisted overrides).
func (h *SettingsHandler) Get(c *gin.Context) {
	merged, err := h.ai.Settings()
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, merged)
}

// Put saves settings and returns the effective (merged) settings.
func (h *SettingsHandler) Put(c *gin.Context) {
	var req model.AISettings
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, NewErrHTTP(http.StatusBadRequest, "设置格式错误"))
		return
	}

	if err := h.store.SaveSettings(req); err != nil {
		respondError(c, err)
		return
	}

	merged, err := h.ai.Settings()
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, merged)
}

// Test validates connectivity using the current (persisted) settings.
func (h *SettingsHandler) Test(c *gin.Context) {
	if err := h.ai.Ping(); err != nil {
		respondError(c, NewErrHTTP(http.StatusBadRequest, err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
