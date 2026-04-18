package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/context-engine/internal/domain/repository"
)

type ProjectHandler struct {
	vectorRepo repository.VectorRepository
}

func NewProjectHandler(vr repository.VectorRepository) *ProjectHandler {
	return &ProjectHandler{vectorRepo: vr}
}

func (h *ProjectHandler) List(c *gin.Context) {
	projects, err := h.vectorRepo.ListProjects()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if projects == nil {
		projects = []*repository.ProjectSummary{}
	}
	c.JSON(http.StatusOK, gin.H{"projects": projects})
}
