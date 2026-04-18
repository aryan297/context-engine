package model

type Function struct {
	ID          string    `json:"id"`
	FileID      string    `json:"file_id"`
	ProjectName string    `json:"project_name"`
	Name        string    `json:"name"`
	Signature   string    `json:"signature"`
	Summary     string    `json:"summary"`
	Embedding   []float32 `json:"embedding,omitempty"`
}
