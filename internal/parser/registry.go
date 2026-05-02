package parser

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

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

// hardSkipDirs are always skipped regardless of .gitignore.
var hardSkipDirs = map[string]bool{
	"node_modules": true,
	".git":         true,
	"vendor":       true,
	".next":        true,
	"dist":         true,
	"build":        true,
	"out":          true,
	"coverage":     true,
	".cache":       true,
	".turbo":       true,
}

// loadGitignoreDirs reads the .gitignore at root and returns a set of
// directory names (or path prefixes) that should be skipped.
func loadGitignoreDirs(root string) map[string]bool {
	skipped := make(map[string]bool)
	f, err := os.Open(filepath.Join(root, ".gitignore"))
	if err != nil {
		return skipped
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Normalise: strip leading/trailing slashes and wildcard-only lines
		line = strings.TrimPrefix(line, "/")
		line = strings.TrimSuffix(line, "/")
		if line == "" || strings.ContainsAny(line, "*?[") {
			continue
		}
		skipped[line] = true
	}
	return skipped
}

// ParseDirectory walks dirPath recursively and parses every supported source file.
// Directories listed in the project's .gitignore are automatically skipped.
func (r *Registry) ParseDirectory(projectID, projectName, dirPath string) ([]*model.File, error) {
	var files []*model.File

	gitignored := loadGitignoreDirs(dirPath)

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			name := info.Name()
			if hardSkipDirs[name] || gitignored[name] {
				return filepath.SkipDir
			}
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
