package neo4j

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"github.com/context-engine/internal/domain/model"
)

type GraphStore struct {
	driver neo4j.DriverWithContext
}

func NewGraphStore(uri, username, password string) (*GraphStore, error) {
	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		return nil, fmt.Errorf("create neo4j driver: %w", err)
	}
	ctx := context.Background()
	if err := driver.VerifyConnectivity(ctx); err != nil {
		return nil, fmt.Errorf("verify neo4j connectivity: %w", err)
	}
	return &GraphStore{driver: driver}, nil
}

func (g *GraphStore) StoreProject(project *model.Project) error {
	ctx := context.Background()
	session := g.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx,
			`MERGE (p:Project {id: $id})
			 SET p.name = $name, p.path = $path`,
			map[string]any{"id": project.ID, "name": project.Name, "path": project.Path},
		)
	})
	return err
}

func (g *GraphStore) StoreFile(file *model.File) error {
	ctx := context.Background()
	session := g.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// Create file node and link to project
		if _, err := tx.Run(ctx,
			`MERGE (f:File {id: $id})
			 SET f.name = $name, f.path = $path, f.summary = $summary
			 WITH f
			 MATCH (p:Project {id: $project_id})
			 MERGE (p)-[:CONTAINS]->(f)`,
			map[string]any{
				"id":         file.ID,
				"name":       file.Name,
				"path":       file.Path,
				"summary":    file.Summary,
				"project_id": file.ProjectID,
			},
		); err != nil {
			return nil, err
		}

		// Link imports as dependency edges between files
		for _, imp := range file.Imports {
			if _, err := tx.Run(ctx,
				`MATCH (f:File {id: $file_id})
				 MERGE (dep:Import {path: $import_path})
				 MERGE (f)-[:IMPORTS]->(dep)`,
				map[string]any{"file_id": file.ID, "import_path": imp},
			); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})
	return err
}

func (g *GraphStore) StoreFunction(fn *model.Function) error {
	ctx := context.Background()
	session := g.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx,
			`MERGE (fn:Function {id: $id})
			 SET fn.name = $name, fn.signature = $signature, fn.summary = $summary
			 WITH fn
			 MATCH (f:File {id: $file_id})
			 MERGE (f)-[:DEFINES]->(fn)`,
			map[string]any{
				"id":        fn.ID,
				"name":      fn.Name,
				"signature": fn.Signature,
				"summary":   fn.Summary,
				"file_id":   fn.FileID,
			},
		)
	})
	return err
}

// GetRelatedNodes fetches files reachable from the given file node within depth hops.
func (g *GraphStore) GetRelatedNodes(fileID string, depth int) ([]*model.File, error) {
	ctx := context.Background()
	session := g.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx,
			fmt.Sprintf(
				`MATCH (start:File {id: $file_id})-[*1..%d]-(related:File)
				 RETURN DISTINCT related.id AS id, related.name AS name,
				        related.path AS path, related.summary AS summary`,
				depth,
			),
			map[string]any{"file_id": fileID},
		)
		if err != nil {
			return nil, err
		}
		var files []*model.File
		for res.Next(ctx) {
			rec := res.Record()
			f := &model.File{
				ID:      stringProp(rec, "id"),
				Name:    stringProp(rec, "name"),
				Path:    stringProp(rec, "path"),
				Summary: stringProp(rec, "summary"),
			}
			files = append(files, f)
		}
		return files, res.Err()
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return result.([]*model.File), nil
}

func stringProp(rec *neo4j.Record, key string) string {
	v, _ := rec.Get(key)
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
