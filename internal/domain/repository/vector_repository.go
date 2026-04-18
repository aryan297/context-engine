package repository

import "github.com/context-engine/internal/domain/model"

type ProjectSummary struct {
	Name      string `json:"name"`
	FileCount int    `json:"file_count"`
	FuncCount int    `json:"func_count"`
}

type VectorRepository interface {
	StoreFileEmbedding(file *model.File) error
	StoreFunctionEmbedding(fn *model.Function) error
	SearchSimilarFiles(embedding []float32, projectName string, topK int) ([]*model.File, error)
	SearchSimilarFunctions(embedding []float32, projectName string, topK int) ([]*model.Function, error)
	ListProjects() ([]*ProjectSummary, error)
}
