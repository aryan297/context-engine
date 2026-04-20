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
	javaImportRe = regexp.MustCompile(`(?m)^import\s+([\w.]+);`)

	// Method: optional annotations, access modifiers, return type, methodName(
	// Captures the method name (last word before the opening paren)
	javaMethodRe = regexp.MustCompile(
		`(?m)(?:(?:public|private|protected|static|final|synchronized|abstract|native|default)\s+)+` +
			`(?:[\w<>\[\],\s]+?\s+)(\w+)\s*\(`,
	)
)

type JavaParser struct{}

func NewJavaParser() *JavaParser { return &JavaParser{} }

func (p *JavaParser) Extensions() []string { return []string{".java"} }

func (p *JavaParser) ParseFile(projectID, projectName, filePath string) (*model.File, error) {
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

	for _, m := range javaImportRe.FindAllStringSubmatch(content, -1) {
		file.Imports = append(file.Imports, m[1])
	}

	seen := map[string]bool{}
	for _, m := range javaMethodRe.FindAllStringSubmatch(content, -1) {
		name := m[1]
		// Skip common false positives like class/interface/enum keywords
		switch name {
		case "class", "interface", "enum", "extends", "implements", "throws":
			continue
		}
		if seen[name] {
			continue
		}
		seen[name] = true
		file.Functions = append(file.Functions, model.Function{
			ID:          uuid.New().String(),
			FileID:      file.ID,
			ProjectName: projectName,
			Name:        name,
			Signature:   name + "(...)",
			Summary:     fmt.Sprintf("Method %s in %s", name, file.Name),
		})
	}

	// Strip package suffix for a readable class name
	className := strings.TrimSuffix(filepath.Base(filePath), ".java")
	file.Summary = fmt.Sprintf("Java class %s with %d methods and %d imports",
		className, len(file.Functions), len(file.Imports))

	return file, nil
}
