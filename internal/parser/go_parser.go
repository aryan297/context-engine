package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/context-engine/internal/domain/model"
	"github.com/google/uuid"
)

type GoParser struct{}

func NewGoParser() *GoParser { return &GoParser{} }

func (p *GoParser) Extensions() []string { return []string{".go"} }

// ParseDirectory walks dirPath recursively and parses every .go file found.
func (p *GoParser) ParseDirectory(projectID, projectName, dirPath string) ([]*model.File, error) {
	var files []*model.File

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		f, parseErr := p.ParseFile(projectID, projectName, path)
		if parseErr != nil {
			return nil // skip unparseable files; don't abort the walk
		}
		files = append(files, f)
		return nil
	})

	return files, err
}

// ParseFile parses a single Go source file and returns its model representation.
func (p *GoParser) ParseFile(projectID, projectName, filePath string) (*model.File, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	file := &model.File{
		ID:          uuid.New().String(),
		ProjectID:   projectID,
		ProjectName: projectName,
		Name:        filepath.Base(filePath),
		Path:        filePath,
	}

	for _, imp := range node.Imports {
		file.Imports = append(file.Imports, strings.Trim(imp.Path.Value, `"`))
	}

	ast.Inspect(node, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}
		function := model.Function{
			ID:          uuid.New().String(),
			FileID:      file.ID,
			ProjectName: projectName,
			Name:        fn.Name.Name,
			Signature:   buildSignature(fn),
			Summary:     fmt.Sprintf("Function %s in %s", fn.Name.Name, file.Name),
		}
		file.Functions = append(file.Functions, function)
		return true
	})

	file.Summary = fmt.Sprintf("Go file %s with %d functions and %d imports",
		file.Name, len(file.Functions), len(file.Imports))

	return file, nil
}

func buildSignature(fn *ast.FuncDecl) string {
	var b strings.Builder
	b.WriteString("func ")

	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		b.WriteString("(")
		if len(fn.Recv.List[0].Names) > 0 {
			b.WriteString(fn.Recv.List[0].Names[0].Name)
		}
		b.WriteString(") ")
	}

	b.WriteString(fn.Name.Name)
	b.WriteString("(")

	if fn.Type.Params != nil {
		params := make([]string, 0, len(fn.Type.Params.List))
		for _, field := range fn.Type.Params.List {
			for range field.Names {
				params = append(params, fmt.Sprintf("%T", field.Type))
			}
			if len(field.Names) == 0 {
				params = append(params, fmt.Sprintf("%T", field.Type))
			}
		}
		b.WriteString(strings.Join(params, ", "))
	}

	b.WriteString(")")

	if fn.Type.Results != nil && len(fn.Type.Results.List) > 0 {
		b.WriteString(" (")
		results := make([]string, len(fn.Type.Results.List))
		for i, r := range fn.Type.Results.List {
			results[i] = fmt.Sprintf("%T", r.Type)
		}
		b.WriteString(strings.Join(results, ", "))
		b.WriteString(")")
	}

	return b.String()
}
