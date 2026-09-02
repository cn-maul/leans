package handler

import (
	"net/http"

	"leans/model"
	"leans/service"

	"github.com/gin-gonic/gin"
)

type AnalyzeHandler struct {
	analyzer *service.Analyzer
}

func NewAnalyzeHandler(analyzer *service.Analyzer) *AnalyzeHandler {
	return &AnalyzeHandler{analyzer: analyzer}
}

func (h *AnalyzeHandler) Handle(c *gin.Context) {
	var req model.AnalyzeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, NewErrHTTP(http.StatusBadRequest, "请求格式错误：题目与科目不能为空"))
		return
	}

	result, err := h.analyzer.Analyze(service.AnalyzeOption{
		Subject:     req.Subject,
		Content:     req.Content,
		SaveHistory: true,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}
