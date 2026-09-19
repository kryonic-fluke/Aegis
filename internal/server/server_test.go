package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kryonic-fluke/aegis/internal/config"
	"github.com/kryonic-fluke/aegis/internal/llm"
	"github.com/kryonic-fluke/aegis/internal/router"
)

type mockProvider struct {
	chatFunc func(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error)
}

func (m *mockProvider) Chat(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
	if m.chatFunc != nil {
		return m.chatFunc(ctx, req)
	}
	return &llm.ChatResponse{
		ID: "mock-id",
		Choices: []llm.Choice{
			{Index: 0, Message: llm.Message{Role: "assistant", Content: "mock content"}},
		},
	}, nil
}

func (m *mockProvider) ChatStream(ctx context.Context, req *llm.ChatRequest) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("data: {}\n\n")), nil
}

type mockEmbedder struct{}

func (m *mockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	return make([]float32, 768), nil
}

func newTestServer() *Server {
	cfg := &config.Config{
		Port:          "8080",
		Env:           "test",
		LogLevel:      "debug",
		ReadTimeout:   5 * time.Second,
		WriteTimeout:  10 * time.Second,
		IdleTimeout:   10 * time.Second,
		ModelFast:     "fast-model",
		ModelBalanced: "balanced-model",
		ModelDeep:     "deep-model",
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	provider := &mockProvider{}
	embedder := &mockEmbedder{}
	modelRouter := router.NewRouter(embedder, cfg)
	_ = modelRouter.InitPrototypes(context.Background())

	return New(cfg, logger, provider, nil, nil, nil, nil, embedder, nil, modelRouter)
}

func TestHealthEndpoint(t *testing.T) {
	srv := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode health response: %v", err)
	}

	if body["env"] != "test" {
		t.Errorf("expected env test, got %s", body["env"])
	}

	if body["status"] != "available" {
		t.Errorf("expected status available, got %s", body["status"])
	}
}

func TestMetricsEndpoint(t *testing.T) {
	srv := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	content := rec.Body.String()
	if !strings.Contains(content, "aegis_requests_total") {
		t.Errorf("expected metrics to contain aegis_requests_total")
	}
}

func TestChatCompletions_RoutingAndProviderHeaders(t *testing.T) {
	srv := newTestServer()

	payload := map[string]interface{}{
		"messages": []map[string]string{
			{"role": "user", "content": "Extract data into JSON"},
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	if routingClass := rec.Header().Get("X-Routing-Class"); routingClass == "" {
		t.Errorf("expected X-Routing-Class header to be set")
	}

	if routingModel := rec.Header().Get("X-Routing-Model"); routingModel == "" {
		t.Errorf("expected X-Routing-Model header to be set")
	}

	if provider := rec.Header().Get("X-Provider"); provider != "primary" {
		t.Errorf("expected X-Provider primary, got %s", provider)
	}
}
