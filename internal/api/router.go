package api

import (
	"github.com/gin-gonic/gin"

	"github.com/context-engine/internal/api/handler"
)

func NewRouter(ingest *handler.IngestHandler, query *handler.QueryHandler) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	v1 := r.Group("/v1")
	{
		v1.POST("/ingest-project", ingest.Handle)
		v1.POST("/query-context", query.Handle)
	}

	return r
}
