package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/context-engine/internal/service"
)

type QueryHandler struct {
	svc *service.QueryService
}

func NewQueryHandler(svc *service.QueryService) *QueryHandler {
	return &QueryHandler{svc: svc}
}

type queryRequest struct {
	ProjectName string `json:"project_name" binding:"required"`
	Query       string `json:"query"        binding:"required"`
}

func (h *QueryHandler) Handle(c *gin.Context) {
	var req queryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.svc.QueryContext(req.ProjectName, req.Query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
