package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/context-engine/internal/service"
)

type IngestHandler struct {
	svc *service.IngestService
}

func NewIngestHandler(svc *service.IngestService) *IngestHandler {
	return &IngestHandler{svc: svc}
}

type ingestRequest struct {
	ProjectName string `json:"project_name" binding:"required"`
	Path        string `json:"path"         binding:"required"`
}

func (h *IngestHandler) Handle(c *gin.Context) {
	var req ingestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.svc.IngestProject(req.ProjectName, req.Path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "project ingested successfully",
		"result":  result,
	})
}
