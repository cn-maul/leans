package handler

import (
	"net/http"

	"leans/storage"

	"github.com/gin-gonic/gin"
)

type StatsHandler struct {
	store *storage.Store
}

func NewStatsHandler(store *storage.Store) *StatsHandler {
	return &StatsHandler{store: store}
}

// Get returns aggregate statistics over all analysis history records.
func (h *StatsHandler) Get(c *gin.Context) {
	stats, err := h.store.GetStats()
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, stats)
}
