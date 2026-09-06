package handler

import (
	"encoding/json"
	"errors"
	"fmt"
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

	result, err := h.analyzer.Analyze(c.Request.Context(), service.AnalyzeOption{
		Subject:     req.Subject,
		Content:     req.Content,
		SaveHistory: true,
	})
	if err != nil {
		if c.Request.Context().Err() != nil {
			// 客户端已断开，写响应无意义。
			return
		}
		// AI 未配置是用户可纠正状态，映射为 4xx 而不是 500。
		if errors.Is(err, service.ErrNotConfigured) {
			respondError(c, NewErrHTTP(http.StatusBadRequest, err.Error()))
			return
		}
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// HandleStream 以 SSE 流式分析：delta 事件转发 AI 内容增量，
// done 事件返回最终规范化结果（与 /analyze 相同结构），error 事件返回错误。
func (h *AnalyzeHandler) HandleStream(c *gin.Context) {
	var req model.AnalyzeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, NewErrHTTP(http.StatusBadRequest, "请求格式错误：题目与科目不能为空"))
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Flush()

	send := func(event string, payload any) {
		data, err := json.Marshal(payload)
		if err != nil {
			return
		}
		fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, data)
		c.Writer.Flush()
	}

	result, err := h.analyzer.AnalyzeStream(c.Request.Context(), service.AnalyzeOption{
		Subject:     req.Subject,
		Content:     req.Content,
		SaveHistory: true,
	}, service.StreamCallbacks{
		OnModel:  func(name string) { send("model", gin.H{"model": name}) },
		OnDelta:  func(text string) { send("delta", gin.H{"t": text}) },
		OnStatus: func(msg string) { send("status", gin.H{"message": msg}) },
	})
	if err != nil {
		if c.Request.Context().Err() != nil {
			// 客户端中止（连接已断开），无法也无需回写错误事件。
			return
		}
		send("error", gin.H{"error": err.Error()})
		return
	}
	send("done", result)
}
