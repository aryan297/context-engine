# Language Support

Context Engine can ingest and index source files from the following languages.
Each language has its own parser in `internal/parser/`.

---

## Supported Languages

| Language   | Extensions        | Parser file               | Extraction method |
|------------|-------------------|---------------------------|-------------------|
| Go         | `.go`             | `go_parser.go`            | AST (`go/ast`)    |
| TypeScript | `.ts`, `.tsx`     | `ts_parser.go`            | Regex             |
| Java       | `.java`           | `java_parser.go`          | Regex             |
| Python     | `.py`             | `python_parser.go`        | Regex             |
| Rust       | `.rs`             | `rust_parser.go`          | Regex             |

---

## What Gets Extracted Per Language

### Go
- **Imports** — all `import` paths
- **Functions** — all `func` declarations including methods (with receiver)
- **Signature** — full parameter and return type list via AST
- **Summary** — `Go file <name> with N functions and N imports`

### TypeScript / TSX
- **Imports** — `import ... from '...'` paths
- **Functions** — `function foo(`, `const foo = (` arrow functions, class methods with access modifiers
- **Signature** — `name(...)`
- **Summary** — `TypeScript file <name> with N functions and N imports`

### Java
- **Imports** — `import x.y.z;` statements
- **Methods** — public / private / protected / static methods
- **Signature** — `name(...)`
- **Summary** — `Java class <ClassName> with N methods and N imports`

### Python
- **Imports** — `import x` and `from x import` statements
- **Functions** — `def` and `async def` functions
- **Classes** — `class Foo` definitions (indexed alongside functions)
- **Signature** — `def name(...)` or `class Name`
- **Summary** — `Python module <name> with N functions/classes and N imports`

### Rust
- **Imports** — `use` statements (root crate/module path)
- **Functions** — `fn`, `pub fn`, `async fn`, `pub async fn`
- **Structs** — `struct Foo` and `pub struct Foo`
- **Enums** — `enum Foo` and `pub enum Foo`
- **Traits** — `trait Foo` and `pub trait Foo`
- **Signature** — `fn name(...)`, `struct Name`, `enum Name`, `trait Name`
- **Summary** — `Rust module <name> with N items and N use statements`

---

## Skipped Directories

The following directories are **always** skipped during ingestion, regardless of language:

| Directory       | Reason                        |
|-----------------|-------------------------------|
| `node_modules`  | npm / yarn packages           |
| `.git`          | Git internals                 |
| `vendor`        | Go vendor directory           |
| `.next`         | Next.js build cache           |
| `dist`          | Compiled output               |
| `build`         | Compiled output               |
| `out`           | Compiled output               |
| `coverage`      | Test coverage reports         |
| `.cache`        | General tool cache            |
| `.turbo`        | Turborepo cache               |
| `__pycache__`   | Python bytecode cache         |
| `.venv` / `venv`| Python virtual environments   |
| `.mypy_cache`   | Mypy type-checker cache       |
| `.pytest_cache` | Pytest cache                  |
| `site-packages` | Installed Python packages     |
| `target`        | Rust build output (`cargo build`) |

In addition, any directory listed in the project's root `.gitignore` is automatically skipped.

---

## Adding a New Language

1. Create `internal/parser/<lang>_parser.go` and implement the `Parser` interface:

```go
type Parser interface {
    Extensions() []string
    ParseFile(projectID, projectName, filePath string) (*model.File, error)
}
```

2. Register it in `internal/parser/registry.go`:

```go
for _, p := range []Parser{
    NewGoParser(),
    NewTypeScriptParser(),
    NewJavaParser(),
    NewPythonParser(),
    NewYourLangParser(), // add here
} {
```

3. If your language has build/cache directories that should be skipped, add them to `hardSkipDirs` in `registry.go`.

That's it — no other changes needed. The registry auto-dispatches by file extension.

---

## Example: Adding a Ruby Parser

```go
// internal/parser/ruby_parser.go
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
    rubyRequireRe = regexp.MustCompile(`(?m)^require(?:_relative)?\s+['"]([^'"]+)['"]`)
    rubyMethodRe  = regexp.MustCompile(`(?m)^\s*def\s+(\w+)`)
    rubyClassRe   = regexp.MustCompile(`(?m)^class\s+(\w+)`)
)

type RubyParser struct{}

func NewRubyParser() *RubyParser { return &RubyParser{} }

func (p *RubyParser) Extensions() []string { return []string{".rb"} }

func (p *RubyParser) ParseFile(projectID, projectName, filePath string) (*model.File, error) {
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

    for _, m := range rubyRequireRe.FindAllStringSubmatch(content, -1) {
        file.Imports = append(file.Imports, m[1])
    }

    seen := map[string]bool{}
    addFn := func(name, sig string) {
        if seen[name] { return }
        seen[name] = true
        file.Functions = append(file.Functions, model.Function{
            ID: uuid.New().String(), FileID: file.ID,
            ProjectName: projectName, Name: name,
            Signature: sig,
            Summary:   fmt.Sprintf("Method %s in %s", name, file.Name),
        })
    }
    for _, m := range rubyMethodRe.FindAllStringSubmatch(content, -1) { addFn(m[1], "def "+m[1]) }
    for _, m := range rubyClassRe.FindAllStringSubmatch(content, -1)  { addFn(m[1], "class "+m[1]) }

    file.Summary = fmt.Sprintf("Ruby file %s with %d methods and %d requires",
        filepath.Base(filePath), len(file.Functions), len(file.Imports))
    return file, nil
}
```

Then add `NewRubyParser()` to the registry and add `".gem"` / `"vendor/bundle"` to `hardSkipDirs` if needed.
