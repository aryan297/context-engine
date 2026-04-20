package parser

import "github.com/context-engine/internal/domain/model"

// Parser can parse source files of a specific language.
type Parser interface {
	Extensions() []string
	ParseFile(projectID, projectName, filePath string) (*model.File, error)
}
