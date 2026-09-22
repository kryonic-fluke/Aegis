package llm

import (
	"context"
	"io"
)

type Provider interface {
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
	ChatStream(ctx context.Context, req *ChatRequest) (io.ReadCloser, error)
}
