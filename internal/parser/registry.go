package parser

import (
	"os"
	"path/filepath"

	"github.com/context-engine/internal/domain/model"
)

// Registry holds all registered parsers and dispatches by file extension.
type Registry struct {
	byExt map[string]Parser
}

// NewRegistry builds a Registry pre-loaded with Go, TypeScript, and Java parsers.
func NewRegistry() *Registry {
	r := &Registry{byExt: make(map[string]Parser)}
	for _, p := range []Parser{NewGoParser(), NewTypeScriptParser(), NewJavaParser()} {
		for _, ext := range p.Extensions() {
			r.byExt[ext] = p
		}
	}
	return r
}

// ParseDirectory walks dirPath recursively and parses every supported source file.
func (r *Registry) ParseDirectory(projectID, projectName, dirPath string) ([]*model.File, error) {
	var files []*model.File

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		p, ok := r.byExt[filepath.Ext(path)]
		if !ok {
			return nil
		}
		f, parseErr := p.ParseFile(projectID, projectName, path)
		if parseErr != nil {
			return nil // skip unparseable files
		}
		files = append(files, f)
		return nil
	})

	return files, err
}
