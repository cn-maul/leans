package handler

import (
	"net/http"

	"leans/subject"

	"github.com/gin-gonic/gin"
)

type SubjectHandler struct {
	store *subject.Store
}

func NewSubjectHandler(store *subject.Store) *SubjectHandler {
	return &SubjectHandler{store: store}
}

func (h *SubjectHandler) List(c *gin.Context) {
	c.JSON(http.StatusOK, h.store.List())
}

func (h *SubjectHandler) GetContent(c *gin.Context) {
	id := c.Param("id")
	sub, err := h.store.Get(id)
	if err != nil {
		respondError(c, NewErrHTTP(http.StatusNotFound, "科目不存在或讲义文件未加载"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":      sub.ID,
		"name":    sub.Name,
		"summary": sub.Summary,
		"tree":    sub.Tree,
	})
}
