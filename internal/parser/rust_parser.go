package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/context-engine/internal/domain/model"
	"github.com/google/uuid"
)

var (
	rustUseRe = regexp.MustCompile(`(?m)^use\s+([\w:]+)`)

	// fn foo(  or  pub fn foo(  or  pub async fn foo(  or  async fn foo(
	rustFnRe = regexp.MustCompile(`(?m)(?:pub\s+)?(?:async\s+)?fn\s+(\w+)\s*[<(]`)

	// struct Foo  or  pub struct Foo
	rustStructRe = regexp.MustCompile(`(?m)(?:pub\s+)?struct\s+(\w+)`)

	// enum Foo  or  pub enum Foo
	rustEnumRe = regexp.MustCompile(`(?m)(?:pub\s+)?enum\s+(\w+)`)

	// trait Foo  or  pub trait Foo
	rustTraitRe = regexp.MustCompile(`(?m)(?:pub\s+)?trait\s+(\w+)`)
)

type RustParser struct{}

func NewRustParser() *RustParser { return &RustParser{} }

func (p *RustParser) Extensions() []string { return []string{".rs"} }

func (p *RustParser) ParseFile(projectID, projectName, filePath string) (*model.File, error) {
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

	// Imports (use statements — take the root crate/module path)
	seen := map[string]bool{}
	for _, m := range rustUseRe.FindAllStringSubmatch(content, -1) {
		imp := m[1]
		if !seen[imp] {
			seen[imp] = true
			file.Imports = append(file.Imports, imp)
		}
	}

	// Functions, structs, enums, traits — all indexed as functions for query
	fnSeen := map[string]bool{}
	add := func(name, sig string) {
		if fnSeen[name] {
			return
		}
		fnSeen[name] = true
		file.Functions = append(file.Functions, model.Function{
			ID:          uuid.New().String(),
			FileID:      file.ID,
			ProjectName: projectName,
			Name:        name,
			Signature:   sig,
			Summary:     fmt.Sprintf("%s in %s", sig, file.Name),
		})
	}

	for _, m := range rustFnRe.FindAllStringSubmatch(content, -1) {
		add(m[1], "fn "+m[1]+"(...)")
	}
	for _, m := range rustStructRe.FindAllStringSubmatch(content, -1) {
		add(m[1], "struct "+m[1])
	}
	for _, m := range rustEnumRe.FindAllStringSubmatch(content, -1) {
		add(m[1], "enum "+m[1])
	}
	for _, m := range rustTraitRe.FindAllStringSubmatch(content, -1) {
		add(m[1], "trait "+m[1])
	}

	moduleName := filepath.Base(filePath)
	file.Summary = fmt.Sprintf("Rust module %s with %d items and %d use statements",
		moduleName, len(file.Functions), len(file.Imports))

	return file, nil
}
