package handler

import (
	"net/http"
	"strings"

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

// Get returns the effective settings (defaults + persisted providers).
func (h *SettingsHandler) Get(c *gin.Context) {
	merged, err := h.ai.Settings()
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, merged)
}

// Put replaces all providers and the active selection, returning the
// normalized settings actually persisted.
func (h *SettingsHandler) Put(c *gin.Context) {
	var req model.AISettings
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, NewErrHTTP(http.StatusBadRequest, "设置格式错误"))
		return
	}

	saved := service.NormalizeSettings(req)
	if err := h.store.SaveSettings(saved); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, saved)
}

// activeRequest 切换主界面二级下拉的当前选择。model 为空表示仅切换
// 供应商，由后端自动选中该供应商下的有效模型。
type activeRequest struct {
	ProviderID string `json:"provider_id" binding:"required"`
	Model      string `json:"model"`
}

// Active 更新激活的供应商与模型。
func (h *SettingsHandler) Active(c *gin.Context) {
	var req activeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, NewErrHTTP(http.StatusBadRequest, "请求格式错误"))
		return
	}

	set, err := h.ai.Settings()
	if err != nil {
		respondError(c, err)
		return
	}
	found := false
	for _, p := range set.Providers {
		if p.ID == req.ProviderID {
			found = true
			break
		}
	}
	if !found {
		respondError(c, NewErrHTTP(http.StatusBadRequest, "供应商不存在，请刷新设置"))
		return
	}

	set.ActiveProviderID = req.ProviderID
	if strings.TrimSpace(req.Model) != "" {
		set.ActiveModel = req.Model
	}
	saved := service.NormalizeSettings(set)
	if err := h.store.SaveSettings(saved); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, saved)
}

// modelsRequest 用表单里的凭据在线获取模型列表，不要求先保存。
type modelsRequest struct {
	BaseURL  string `json:"base_url" binding:"required"`
	APIKey   string `json:"api_key"`
	Protocol string `json:"protocol"`
}

// Models 调用服务端 GET /models 拉取模型 ID 列表。
func (h *SettingsHandler) Models(c *gin.Context) {
	var req modelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, NewErrHTTP(http.StatusBadRequest, "请求格式错误：需要 Base URL"))
		return
	}
	models, err := h.ai.FetchModels(c.Request.Context(), req.BaseURL, req.APIKey, req.Protocol)
	if err != nil {
		respondError(c, NewErrHTTP(http.StatusBadRequest, err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"models": models})
}

// testRequest 允许在保存前测试表单中的供应商配置；为空则测试当前
// 激活的供应商。
type testRequest struct {
	Provider *model.AIProvider `json:"provider"`
	Model    string            `json:"model"`
}

// Test validates connectivity against the given (or active) provider.
func (h *SettingsHandler) Test(c *gin.Context) {
	var req testRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Provider == nil {
		if err := h.ai.Ping(); err != nil {
			respondError(c, NewErrHTTP(http.StatusBadRequest, err.Error()))
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}
	if err := h.ai.PingProvider(*req.Provider, req.Model); err != nil {
		respondError(c, NewErrHTTP(http.StatusBadRequest, err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
