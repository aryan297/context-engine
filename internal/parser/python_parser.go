package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/context-engine/internal/domain/model"
	"github.com/google/uuid"
)

var (
	pyImportRe = regexp.MustCompile(`(?m)^(?:import\s+([\w.]+)|from\s+([\w.]+)\s+import)`)

	// def foo(  or  async def foo(
	pyFuncRe = regexp.MustCompile(`(?m)^[ \t]*(?:async\s+)?def\s+(\w+)\s*\(`)

	// class Foo:  or  class Foo(Base):
	pyClassRe = regexp.MustCompile(`(?m)^class\s+(\w+)\s*[:(]`)
)

type PythonParser struct{}

func NewPythonParser() *PythonParser { return &PythonParser{} }

func (p *PythonParser) Extensions() []string { return []string{".py"} }

func (p *PythonParser) ParseFile(projectID, projectName, filePath string) (*model.File, error) {
	src, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	content := string(src)

	file := &model.File{
		ID:          uuid.New().String(),
		ProjectID:   projectID,
		ProjectName: projectName,
		Name:        filepath.Base(filePath),
		Path:        filePath,
	}

	// Imports
	seen := map[string]bool{}
	for _, m := range pyImportRe.FindAllStringSubmatch(content, -1) {
		imp := m[1]
		if imp == "" {
			imp = m[2]
		}
		if !seen[imp] {
			seen[imp] = true
			file.Imports = append(file.Imports, imp)
		}
	}

	// Functions and methods
	fnSeen := map[string]bool{}
	for _, m := range pyFuncRe.FindAllStringSubmatch(content, -1) {
		name := m[1]
		if fnSeen[name] {
			continue
		}
		fnSeen[name] = true
		file.Functions = append(file.Functions, model.Function{
			ID:          uuid.New().String(),
			FileID:      file.ID,
			ProjectName: projectName,
			Name:        name,
			Signature:   "def " + name + "(...)",
			Summary:     fmt.Sprintf("Function %s in %s", name, file.Name),
		})
	}

	// Classes — add as functions so they appear in query results
	for _, m := range pyClassRe.FindAllStringSubmatch(content, -1) {
		name := m[1]
		if fnSeen[name] {
			continue
		}
		fnSeen[name] = true
		file.Functions = append(file.Functions, model.Function{
			ID:          uuid.New().String(),
			FileID:      file.ID,
			ProjectName: projectName,
			Name:        name,
			Signature:   "class " + name,
			Summary:     fmt.Sprintf("Class %s in %s", name, file.Name),
		})
	}

	moduleName := strings.TrimSuffix(filepath.Base(filePath), ".py")
	file.Summary = fmt.Sprintf("Python module %s with %d functions/classes and %d imports",
		moduleName, len(file.Functions), len(file.Imports))

	return file, nil
}
