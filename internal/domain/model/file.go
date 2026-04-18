package model

type File struct {
	ID          string     `json:"id"`
	ProjectID   string     `json:"project_id"`
	ProjectName string     `json:"project_name"`
	Name        string     `json:"name"`
	Path        string     `json:"path"`
	Imports     []string   `json:"imports"`
	Summary     string     `json:"summary"`
	Embedding   []float32  `json:"embedding,omitempty"`
	Functions   []Function `json:"functions"`
}
