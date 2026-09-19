package router

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/kryonic-fluke/aegis/internal/config"
	"github.com/kryonic-fluke/aegis/internal/llm"
)

type RouteClass string

const (
	RouteFast     RouteClass = "FAST"
	RouteBalanced RouteClass = "BALANCED"
	RouteDeep     RouteClass = "DEEP"
)

const (
	prototypeFast     = "Extract parse convert JSON format entity classification simple lookup basic summarization short translation straightforward factual question and answer"
	prototypeBalanced = "Write clean code debug syntax errors explain functions algorithms step-by-step tutorial compare technologies implement feature refactor"
	prototypeDeep     = "Architect high-throughput distributed systems complex multi-step reasoning mathematical proofs deep concurrency analysis memory leak diagnostics raft consensus failure domain design formal verification"
)

type Decision struct {
	Class           RouteClass
	SelectedModel   string
	SemanticScores  map[RouteClass]float64
	HeuristicScores map[RouteClass]float64
	FinalScores     map[RouteClass]float64
}

type Router struct {
	embedder    llm.Embedder
	cfg         *config.Config
	fastVec     []float32
	balancedVec []float32
	deepVec     []float32
	mu          sync.RWMutex
}

func NewRouter(embedder llm.Embedder, cfg *config.Config) *Router {
	return &Router{
		embedder: embedder,
		cfg:      cfg,
	}
}

func (r *Router) InitPrototypes(ctx context.Context) error {
	if r.embedder == nil {
		return fmt.Errorf("embedder is nil, cannot initialize prototypes")
	}

	fVec, err := r.embedder.Embed(ctx, prototypeFast)
	if err != nil {
		return fmt.Errorf("failed to embed fast prototype: %w", err)
	}

	bVec, err := r.embedder.Embed(ctx, prototypeBalanced)
	if err != nil {
		return fmt.Errorf("failed to embed balanced prototype: %w", err)
	}

	dVec, err := r.embedder.Embed(ctx, prototypeDeep)
	if err != nil {
		return fmt.Errorf("failed to embed deep prototype: %w", err)
	}

	r.mu.Lock()
	r.fastVec = fVec
	r.balancedVec = bVec
	r.deepVec = dVec
	r.mu.Unlock()

	return nil
}

func (r *Router) Route(ctx context.Context, prompt string, embedding []float32) (*Decision, error) {
	if embedding == nil && r.embedder != nil && prompt != "" {
		emb, err := r.embedder.Embed(ctx, prompt)
		if err == nil {
			embedding = emb
		}
	}

	r.mu.RLock()
	fVec := r.fastVec
	bVec := r.balancedVec
	dVec := r.deepVec
	r.mu.RUnlock()

	semScores := map[RouteClass]float64{
		RouteFast:     0.0,
		RouteBalanced: 0.0,
		RouteDeep:     0.0,
	}

	if len(embedding) > 0 && len(fVec) == len(embedding) {
		semScores[RouteFast] = cosineSimilarity(embedding, fVec)
		semScores[RouteBalanced] = cosineSimilarity(embedding, bVec)
		semScores[RouteDeep] = cosineSimilarity(embedding, dVec)
	}

	heurScores := computeHeuristics(prompt)

	finalScores := map[RouteClass]float64{
		RouteFast:     (0.65 * semScores[RouteFast]) + (0.35 * heurScores[RouteFast]),
		RouteBalanced: (0.65 * semScores[RouteBalanced]) + (0.35 * heurScores[RouteBalanced]),
		RouteDeep:     (0.65 * semScores[RouteDeep]) + (0.35 * heurScores[RouteDeep]),
	}

	selectedClass := RouteFast
	if finalScores[RouteDeep] > finalScores[RouteBalanced] && finalScores[RouteDeep] > finalScores[RouteFast] {
		selectedClass = RouteDeep
	} else if finalScores[RouteBalanced] > finalScores[RouteFast] {
		selectedClass = RouteBalanced
	}

	selectedModel := r.cfg.ModelFast
	switch selectedClass {
	case RouteBalanced:
		selectedModel = r.cfg.ModelBalanced
	case RouteDeep:
		selectedModel = r.cfg.ModelDeep
	}

	return &Decision{
		Class:           selectedClass,
		SelectedModel:   selectedModel,
		SemanticScores:  semScores,
		HeuristicScores: heurScores,
		FinalScores:     finalScores,
	}, nil
}

func computeHeuristics(prompt string) map[RouteClass]float64 {
	scores := map[RouteClass]float64{
		RouteFast:     0.1,
		RouteBalanced: 0.1,
		RouteDeep:     0.1,
	}

	lower := strings.ToLower(prompt)
	length := len(prompt)

	if length < 120 {
		scores[RouteFast] += 0.3
	} else if length > 1500 {
		scores[RouteDeep] += 0.35
	}

	fastKeywords := []string{"extract", "json", "format", "list", "convert", "table", "csv", "summarize in one sentence", "translate"}
	for _, kw := range fastKeywords {
		if strings.Contains(lower, kw) {
			scores[RouteFast] += 0.3
			break
		}
	}

	balancedKeywords := []string{"code", "function", "debug", "implement", "explain", "refactor", "how to", "write a test"}
	for _, kw := range balancedKeywords {
		if strings.Contains(lower, kw) {
			scores[RouteBalanced] += 0.3
			break
		}
	}

	deepKeywords := []string{"architect", "tradeoffs", "distributed", "concurrency", "consensus", "byzantine", "raft", "failure domain", "system design", "memory leak", "formal verification", "mathematical proof"}
	for _, kw := range deepKeywords {
		if strings.Contains(lower, kw) {
			scores[RouteDeep] += 0.4
			break
		}
	}

	return scores
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0.0
	}

	var dot, normA, normB float64
	for i := 0; i < len(a); i++ {
		valA := float64(a[i])
		valB := float64(b[i])
		dot += valA * valB
		normA += valA * valA
		normB += valB * valB
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
