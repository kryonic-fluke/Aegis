package llm

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
)

type testProvider struct {
	chatFunc func(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
}

func (t *testProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	return t.chatFunc(ctx, req)
}

func (t *testProvider) ChatStream(ctx context.Context, req *ChatRequest) (io.ReadCloser, error) {
	return nil, nil
}

func TestFailoverProvider_PrimarySuccess(t *testing.T) {
	primary := &testProvider{
		chatFunc: func(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
			return &ChatResponse{ID: "primary-resp"}, nil
		},
	}
	secondary := &testProvider{
		chatFunc: func(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
			t.Fatal("secondary should not be called when primary succeeds")
			return nil, nil
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	failover := NewFailoverProvider(primary, secondary, logger)

	ctx, info := WithFailoverTracking(context.Background())
	resp, err := failover.Chat(ctx, &ChatRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.ID != "primary-resp" {
		t.Errorf("expected primary-resp, got %s", resp.ID)
	}

	if info.Triggered {
		t.Errorf("failover should not have been triggered")
	}
}

func TestFailoverProvider_PrimaryRetryableFailure(t *testing.T) {
	primary := &testProvider{
		chatFunc: func(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
			return nil, errors.New("upstream status 503 Service Unavailable")
		},
	}
	secondary := &testProvider{
		chatFunc: func(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
			return &ChatResponse{ID: "secondary-resp"}, nil
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	failover := NewFailoverProvider(primary, secondary, logger)

	ctx, info := WithFailoverTracking(context.Background())
	resp, err := failover.Chat(ctx, &ChatRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.ID != "secondary-resp" {
		t.Errorf("expected secondary-resp, got %s", resp.ID)
	}

	if !info.Triggered {
		t.Errorf("failover should have been triggered")
	}
}
