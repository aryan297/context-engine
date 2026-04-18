package repository

import "github.com/context-engine/internal/domain/model"

type GraphRepository interface {
	StoreProject(project *model.Project) error
	StoreFile(file *model.File) error
	StoreFunction(fn *model.Function) error
	// GetRelatedNodes fetches files related to a given file node up to the given depth.
	GetRelatedNodes(fileID string, depth int) ([]*model.File, error)
}
