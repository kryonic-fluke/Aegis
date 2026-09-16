package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kryonic-fluke/aegis/internal/llm"
)

type SemanticCache struct {
	pool *pgxpool.Pool
}

func NewSemanticCache(pool *pgxpool.Pool) *SemanticCache {
	return &SemanticCache{
		pool: pool,
	}
}

func (c *SemanticCache) InitSchema(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS semantic_cache (
		id BIGSERIAL PRIMARY KEY,
		prompt TEXT NOT NULL,
		model VARCHAR(100) NOT NULL,
		response JSONB NOT NULL,
		embedding vector(768) NOT NULL,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_semantic_cache_embedding 
	ON semantic_cache USING hnsw (embedding vector_cosine_ops);
	`
	_, err := c.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to initialize semantic cache schema: %w", err)
	}
	return nil
}

func (c *SemanticCache) FindSimilar(ctx context.Context, model string, embedding []float32, threshold float64) (*llm.ChatResponse, float64, error) {
	maxDistance := 1.0 - threshold
	vectorStr := formatVector(embedding)

	query := `
	SELECT response, 1 - (embedding <=> $1) AS similarity
	FROM semantic_cache
	WHERE model = $2 AND (embedding <=> $1) <= $3
	ORDER BY embedding <=> $1 ASC
	LIMIT 1;
	`

	var responseJSON []byte
	var similarity float64
	err := c.pool.QueryRow(ctx, query, vectorStr, model, maxDistance).Scan(&responseJSON, &similarity)
	if err == pgx.ErrNoRows {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query semantic cache: %w", err)
	}

	var resp llm.ChatResponse
	if err := json.Unmarshal(responseJSON, &resp); err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal semantic response: %w", err)
	}

	return &resp, similarity, nil
}

func (c *SemanticCache) Store(ctx context.Context, prompt string, model string, embedding []float32, resp *llm.ChatResponse) error {
	vectorStr := formatVector(embedding)
	responseJSON, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal response for semantic store: %w", err)
	}

	query := `
	INSERT INTO semantic_cache (prompt, model, response, embedding)
	VALUES ($1, $2, $3, $4);
	`
	_, err = c.pool.Exec(ctx, query, prompt, model, responseJSON, vectorStr)
	if err != nil {
		return fmt.Errorf("failed to insert into semantic cache: %w", err)
	}
	return nil
}

func formatVector(v []float32) string {
	var sb strings.Builder
	sb.WriteByte('[')
	for i, val := range v {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(strconv.FormatFloat(float64(val), 'f', 6, 32))
	}
	sb.WriteByte(']')
	return sb.String()
}
