package service

import (
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/context-engine/internal/domain/model"
	"github.com/context-engine/internal/domain/repository"
	"github.com/context-engine/internal/embedding"
	"github.com/context-engine/internal/utils"
)

const (
	topKFiles     = 5
	topKFunctions = 10
	graphDepth    = 2
	cacheTTL      = 10 * time.Minute
)

type QueryService struct {
	graphRepo  repository.GraphRepository
	vectorRepo repository.VectorRepository
	cacheRepo  repository.CacheRepository
	embedder   embedding.Embedder
}

func NewQueryService(
	gr repository.GraphRepository,
	vr repository.VectorRepository,
	cr repository.CacheRepository,
	emb embedding.Embedder,
) *QueryService {
	return &QueryService{
		graphRepo:  gr,
		vectorRepo: vr,
		cacheRepo:  cr,
		embedder:   emb,
	}
}

type QueryResult struct {
	Query     string           `json:"query"`
	Files     []*model.File    `json:"files"`
	Functions []*model.Function `json:"functions"`
	Related   []*model.File    `json:"related_files"`
}

func (s *QueryService) QueryContext(projectName, query string) (*QueryResult, error) {
	log := utils.Get()
	cacheKey := fmt.Sprintf("ctx:%s:%s", projectName, query)

	if cached, err := s.cacheRepo.Get(cacheKey); err == nil {
		var result QueryResult
		if json.Unmarshal([]byte(cached), &result) == nil {
			log.Info("cache hit", zap.String("key", cacheKey))
			return &result, nil
		}
	}

	emb := s.embedder.Generate(query)

	files, err := s.vectorRepo.SearchSimilarFiles(emb, projectName, topKFiles)
	if err != nil {
		return nil, fmt.Errorf("vector search files: %w", err)
	}

	fns, err := s.vectorRepo.SearchSimilarFunctions(emb, projectName, topKFunctions)
	if err != nil {
		return nil, fmt.Errorf("vector search functions: %w", err)
	}

	// Expand context via graph relationships for each matched file
	seen := make(map[string]bool)
	var related []*model.File
	for _, f := range files {
		neighbors, err := s.graphRepo.GetRelatedNodes(f.ID, graphDepth)
		if err != nil {
			log.Warn("graph expand failed", zap.String("file", f.ID), zap.Error(err))
			continue
		}
		for _, n := range neighbors {
			if !seen[n.ID] {
				seen[n.ID] = true
				related = append(related, n)
			}
		}
	}

	result := &QueryResult{
		Query:     query,
		Files:     files,
		Functions: fns,
		Related:   related,
	}

	if data, err := json.Marshal(result); err == nil {
		_ = s.cacheRepo.Set(cacheKey, string(data), cacheTTL)
	}

	return result, nil
}
