package llm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
)

type failoverContextKey struct{}

type FailoverInfo struct {
	Triggered bool
	Error     error
}

func WithFailoverTracking(ctx context.Context) (context.Context, *FailoverInfo) {
	info := &FailoverInfo{}
	return context.WithValue(ctx, failoverContextKey{}, info), info
}

type FailoverProvider struct {
	primary   Provider
	secondary Provider
	logger    *slog.Logger
}

func NewFailoverProvider(primary, secondary Provider, logger *slog.Logger) *FailoverProvider {
	return &FailoverProvider{
		primary:   primary,
		secondary: secondary,
		logger:    logger,
	}
}

func (f *FailoverProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	resp, err := f.primary.Chat(ctx, req)
	if err == nil {
		return resp, nil
	}

	if f.secondary == nil || !isRetryable(err) {
		return nil, err
	}

	f.logger.Warn("primary provider failed, executing failover", "error", err)

	if info, ok := ctx.Value(failoverContextKey{}).(*FailoverInfo); ok && info != nil {
		info.Triggered = true
		info.Error = err
	}

	resp, secErr := f.secondary.Chat(ctx, req)
	if secErr != nil {
		return nil, fmt.Errorf("primary failed: %v; secondary failover failed: %w", err, secErr)
	}

	return resp, nil
}

func (f *FailoverProvider) ChatStream(ctx context.Context, req *ChatRequest) (io.ReadCloser, error) {
	stream, err := f.primary.ChatStream(ctx, req)
	if err == nil {
		return stream, nil
	}

	if f.secondary == nil || !isRetryable(err) {
		return nil, err
	}

	f.logger.Warn("primary stream failed, executing failover", "error", err)

	if info, ok := ctx.Value(failoverContextKey{}).(*FailoverInfo); ok && info != nil {
		info.Triggered = true
		info.Error = err
	}

	stream, secErr := f.secondary.ChatStream(ctx, req)
	if secErr != nil {
		return nil, fmt.Errorf("primary stream failed: %v; secondary failover failed: %w", err, secErr)
	}

	return stream, nil
}

func isRetryable(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}

	msg := strings.ToLower(err.Error())
	retryablePatterns := []string{
		"status 429",
		"status 500",
		"status 502",
		"status 503",
		"status 504",
		"resource_exhausted",
		"rate limit",
		"connection reset",
		"timeout",
		"eof",
	}

	for _, p := range retryablePatterns {
		if strings.Contains(msg, p) {
			return true
		}
	}

	return false
}
