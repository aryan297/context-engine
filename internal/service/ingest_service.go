package service

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/context-engine/internal/domain/model"
	"github.com/context-engine/internal/domain/repository"
	"github.com/context-engine/internal/embedding"
	"github.com/context-engine/internal/parser"
	"github.com/context-engine/internal/utils"
	"github.com/google/uuid"
)

type IngestService struct {
	parser     *parser.Registry
	graphRepo  repository.GraphRepository
	vectorRepo repository.VectorRepository
	embedder   embedding.Embedder
}

func NewIngestService(
	p *parser.Registry,
	gr repository.GraphRepository,
	vr repository.VectorRepository,
	emb embedding.Embedder,
) *IngestService {
	return &IngestService{
		parser:     p,
		graphRepo:  gr,
		vectorRepo: vr,
		embedder:   emb,
	}
}

type IngestResult struct {
	ProjectID     string `json:"project_id"`
	FilesIngested int    `json:"files_ingested"`
	FuncsIngested int    `json:"funcs_ingested"`
}

func (s *IngestService) IngestProject(projectName, path string) (*IngestResult, error) {
	log := utils.Get()

	project := &model.Project{
		ID:   uuid.New().String(),
		Name: projectName,
		Path: path,
	}

	if err := s.graphRepo.StoreProject(project); err != nil {
		return nil, fmt.Errorf("store project in graph: %w", err)
	}
	log.Info("project node created", zap.String("project", projectName))

	files, err := s.parser.ParseDirectory(project.ID, projectName, path)
	if err != nil {
		return nil, fmt.Errorf("parse directory: %w", err)
	}

	result := &IngestResult{ProjectID: project.ID}

	for _, file := range files {
		file.Embedding = s.embedder.Generate(file.Summary)

		if err := s.graphRepo.StoreFile(file); err != nil {
			log.Warn("graph store file failed", zap.String("file", file.Path), zap.Error(err))
			continue
		}
		if err := s.vectorRepo.StoreFileEmbedding(file); err != nil {
			log.Warn("vector store file failed", zap.String("file", file.Path), zap.Error(err))
			continue
		}
		result.FilesIngested++

		for i := range file.Functions {
			fn := &file.Functions[i]
			fn.Embedding = s.embedder.Generate(fn.Summary)

			if err := s.graphRepo.StoreFunction(fn); err != nil {
				log.Warn("graph store function failed", zap.String("fn", fn.Name), zap.Error(err))
				continue
			}
			if err := s.vectorRepo.StoreFunctionEmbedding(fn); err != nil {
				log.Warn("vector store function failed", zap.String("fn", fn.Name), zap.Error(err))
				continue
			}
			result.FuncsIngested++
		}
		log.Info("ingested file", zap.String("file", file.Name), zap.Int("functions", len(file.Functions)))
	}

	return result, nil
}
