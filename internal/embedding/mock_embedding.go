package embedding

import (
	"math"
	"math/rand"
	"time"
)

const Dimensions = 1536

type Embedder interface {
	Generate(text string) []float32
}

type MockEmbedder struct {
	rng *rand.Rand
}

func NewMockEmbedder() *MockEmbedder {
	return &MockEmbedder{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Generate returns a normalized random unit vector of length Dimensions.
// In production, replace with a real embedding API call.
func (e *MockEmbedder) Generate(_ string) []float32 {
	vec := make([]float32, Dimensions)
	var sumSq float64
	for i := range vec {
		v := e.rng.Float32()*2 - 1
		vec[i] = v
		sumSq += float64(v) * float64(v)
	}
	norm := float32(math.Sqrt(sumSq))
	for i := range vec {
		vec[i] /= norm
	}
	return vec
}
