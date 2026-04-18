package postgres

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	_ "github.com/lib/pq"

	"github.com/context-engine/internal/domain/model"
)

type VectorStore struct {
	db *sql.DB
}

func NewVectorStore(dsn string) (*VectorStore, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	store := &VectorStore{db: db}
	if err := store.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return store, nil
}

func (s *VectorStore) migrate() error {
	_, err := s.db.Exec(`
		CREATE EXTENSION IF NOT EXISTS vector;

		CREATE TABLE IF NOT EXISTS files (
			id           TEXT PRIMARY KEY,
			project_name TEXT NOT NULL,
			name         TEXT NOT NULL,
			path         TEXT NOT NULL,
			summary      TEXT,
			embedding    vector(1536)
		);

		CREATE TABLE IF NOT EXISTS functions (
			id           TEXT PRIMARY KEY,
			file_id      TEXT REFERENCES files(id) ON DELETE CASCADE,
			project_name TEXT NOT NULL,
			name         TEXT NOT NULL,
			signature    TEXT,
			summary      TEXT,
			embedding    vector(1536)
		);

		CREATE INDEX IF NOT EXISTS files_embedding_idx
			ON files USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

		CREATE INDEX IF NOT EXISTS functions_embedding_idx
			ON functions USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
	`)
	return err
}

func (s *VectorStore) StoreFileEmbedding(file *model.File) error {
	_, err := s.db.Exec(
		`INSERT INTO files (id, project_name, name, path, summary, embedding)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (id) DO UPDATE
		   SET summary = EXCLUDED.summary, embedding = EXCLUDED.embedding`,
		file.ID, file.ProjectName, file.Name, file.Path, file.Summary,
		vectorLiteral(file.Embedding),
	)
	return err
}

func (s *VectorStore) StoreFunctionEmbedding(fn *model.Function) error {
	_, err := s.db.Exec(
		`INSERT INTO functions (id, file_id, project_name, name, signature, summary, embedding)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (id) DO UPDATE
		   SET summary = EXCLUDED.summary, embedding = EXCLUDED.embedding`,
		fn.ID, fn.FileID, fn.ProjectName, fn.Name, fn.Signature, fn.Summary,
		vectorLiteral(fn.Embedding),
	)
	return err
}

func (s *VectorStore) SearchSimilarFiles(embedding []float32, projectName string, topK int) ([]*model.File, error) {
	rows, err := s.db.Query(
		`SELECT id, project_name, name, path, summary
		 FROM files
		 WHERE project_name = $1
		 ORDER BY embedding <=> $2::vector
		 LIMIT $3`,
		projectName, vectorLiteral(embedding), topK,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*model.File
	for rows.Next() {
		f := &model.File{}
		if err := rows.Scan(&f.ID, &f.ProjectName, &f.Name, &f.Path, &f.Summary); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

func (s *VectorStore) SearchSimilarFunctions(embedding []float32, projectName string, topK int) ([]*model.Function, error) {
	rows, err := s.db.Query(
		`SELECT id, file_id, project_name, name, signature, summary
		 FROM functions
		 WHERE project_name = $1
		 ORDER BY embedding <=> $2::vector
		 LIMIT $3`,
		projectName, vectorLiteral(embedding), topK,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fns []*model.Function
	for rows.Next() {
		fn := &model.Function{}
		if err := rows.Scan(&fn.ID, &fn.FileID, &fn.ProjectName, &fn.Name, &fn.Signature, &fn.Summary); err != nil {
			return nil, err
		}
		fns = append(fns, fn)
	}
	return fns, rows.Err()
}

// vectorLiteral converts a float32 slice to pgvector literal format: '[1.0,2.0,...]'
func vectorLiteral(v []float32) string {
	parts := make([]string, len(v))
	for i, val := range v {
		parts[i] = strconv.FormatFloat(float64(val), 'f', 8, 32)
	}
	return "[" + strings.Join(parts, ",") + "]"
}
