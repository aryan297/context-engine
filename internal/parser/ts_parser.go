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
	tsImportRe = regexp.MustCompile(`(?m)^import\s+.*?from\s+['"]([^'"]+)['"]`)

	// named function declarations: function foo(
	tsFuncDeclRe = regexp.MustCompile(`(?m)(?:export\s+)?(?:async\s+)?function\s+(\w+)\s*\(`)

	// arrow / const functions: const foo = (...) =>  or  const foo = async (...) =>
	tsArrowRe = regexp.MustCompile(`(?m)(?:export\s+)?const\s+(\w+)\s*=\s*(?:async\s*)?\(`)

	// class methods: public/private/protected/async methodName(
	tsMethodRe = regexp.MustCompile(`(?m)(?:(?:public|private|protected|static|async)\s+)+(\w+)\s*\(`)
)

type TypeScriptParser struct{}

func NewTypeScriptParser() *TypeScriptParser { return &TypeScriptParser{} }

func (p *TypeScriptParser) Extensions() []string { return []string{".ts", ".tsx"} }

func (p *TypeScriptParser) ParseFile(projectID, projectName, filePath string) (*model.File, error) {
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
	for _, m := range tsImportRe.FindAllStringSubmatch(content, -1) {
		file.Imports = append(file.Imports, m[1])
	}

	// Functions — collect unique names
	seen := map[string]bool{}
	addFunc := func(name string) {
		if seen[name] || strings.HasPrefix(name, "_") {
			return
		}
		seen[name] = true
		file.Functions = append(file.Functions, model.Function{
			ID:          uuid.New().String(),
			FileID:      file.ID,
			ProjectName: projectName,
			Name:        name,
			Signature:   name + "(...)",
			Summary:     fmt.Sprintf("Function %s in %s", name, file.Name),
		})
	}

	for _, m := range tsFuncDeclRe.FindAllStringSubmatch(content, -1) {
		addFunc(m[1])
	}
	for _, m := range tsArrowRe.FindAllStringSubmatch(content, -1) {
		addFunc(m[1])
	}
	for _, m := range tsMethodRe.FindAllStringSubmatch(content, -1) {
		addFunc(m[1])
	}

	file.Summary = fmt.Sprintf("TypeScript file %s with %d functions and %d imports",
		file.Name, len(file.Functions), len(file.Imports))

	return file, nil
}
