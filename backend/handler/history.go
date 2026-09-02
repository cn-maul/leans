package handler

import (
	"strconv"

	"net/http"

	"leans/model"
	"leans/storage"

	"github.com/gin-gonic/gin"
)

type HistoryHandler struct {
	store *storage.Store
}

func NewHistoryHandler(store *storage.Store) *HistoryHandler {
	return &HistoryHandler{store: store}
}

// List returns recent analysis history (summary fields only).
func (h *HistoryHandler) List(c *gin.Context) {
	limit := 50
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}

	items, err := h.store.ListHistory(limit)
	if err != nil {
		respondError(c, err)
		return
	}

	if items == nil {
		items = []model.HistoryItem{}
	}
	c.JSON(http.StatusOK, items)
}

// Get returns a single history record including the stored analysis result.
func (h *HistoryHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		respondError(c, NewErrHTTP(http.StatusBadRequest, "无效的历史记录 ID"))
		return
	}

	item, err := h.store.GetHistory(id)
	if err != nil {
		respondError(c, NewErrHTTP(http.StatusNotFound, "历史记录不存在"))
		return
	}
	c.JSON(http.StatusOK, item)
}

// Clear deletes all history records.
func (h *HistoryHandler) Clear(c *gin.Context) {
	if err := h.store.ClearHistory(); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
