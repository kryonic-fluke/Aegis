package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/kryonic-fluke/aegis/internal/llm"
	"github.com/kryonic-fluke/aegis/internal/router"
)

func (s *Server) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	var chatReq llm.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&chatReq); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if chatReq.Stream {
		s.handleChatStream(w, r, &chatReq)
		return
	}

	s.handleChatNonStream(w, r, &chatReq)
}

func (s *Server) handleChatNonStream(w http.ResponseWriter, r *http.Request, req *llm.ChatRequest) {
	var cacheKey string
	if s.exactCache != nil {
		key, err := s.exactCache.GenerateKey(req)
		if err == nil {
			cacheKey = key
			cachedResp, err := s.exactCache.Get(r.Context(), cacheKey)
			if err == nil && cachedResp != nil {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Cache", "EXACT_HIT")
				w.WriteHeader(http.StatusOK)
				if err := json.NewEncoder(w).Encode(cachedResp); err != nil {
					s.logger.Error("failed to encode cached chat response", "error", err)
				}
				return
			}
		}
	}

	prompt := extractPrompt(req.Messages)
	var promptEmbedding []float32

	if s.semanticCache != nil && s.embedder != nil && prompt != "" {
		emb, err := s.embedder.Embed(r.Context(), prompt)
		if err == nil {
			promptEmbedding = emb
			semanticResp, similarity, err := s.semanticCache.FindSimilar(r.Context(), req.Model, promptEmbedding, s.cfg.SemanticCacheThreshold)
			if err == nil && semanticResp != nil {
				if s.exactCache != nil && cacheKey != "" {
					_ = s.exactCache.Set(r.Context(), cacheKey, semanticResp)
				}
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Cache", "SEMANTIC_HIT")
				w.Header().Set("X-Cache-Similarity", fmt.Sprintf("%.4f", similarity))
				w.WriteHeader(http.StatusOK)
				if err := json.NewEncoder(w).Encode(semanticResp); err != nil {
					s.logger.Error("failed to encode semantic cached response", "error", err)
				}
				return
			}
		} else {
			s.logger.Warn("failed to generate prompt embedding for cache", "error", err)
		}
	}

	var routingDecision *router.Decision
	if s.modelRouter != nil && prompt != "" {
		dec, err := s.modelRouter.Route(r.Context(), prompt, promptEmbedding)
		if err == nil && dec != nil {
			routingDecision = dec
			req.Model = dec.SelectedModel
			w.Header().Set("X-Routing-Class", string(dec.Class))
			w.Header().Set("X-Routing-Model", dec.SelectedModel)
			w.Header().Set("X-Routing-Scores", fmt.Sprintf("FAST=%.2f,BALANCED=%.2f,DEEP=%.2f",
				dec.FinalScores[router.RouteFast],
				dec.FinalScores[router.RouteBalanced],
				dec.FinalScores[router.RouteDeep]))
		} else {
			s.logger.Warn("model routing failed, using default model", "error", err)
		}
	}

	resp, err := s.provider.Chat(r.Context(), req)
	if err != nil {
		s.logger.Error("provider chat failed", "error", err)
		http.Error(w, fmt.Sprintf("provider error: %v", err), http.StatusBadGateway)
		return
	}

	if s.exactCache != nil && cacheKey != "" {
		if err := s.exactCache.Set(r.Context(), cacheKey, resp); err != nil {
			s.logger.Error("failed to exact cache chat response", "error", err)
		}
	}

	if s.semanticCache != nil && prompt != "" {
		if promptEmbedding == nil && s.embedder != nil {
			emb, err := s.embedder.Embed(r.Context(), prompt)
			if err == nil {
				promptEmbedding = emb
			}
		}
		if promptEmbedding != nil {
			if err := s.semanticCache.Store(r.Context(), prompt, req.Model, promptEmbedding, resp); err != nil {
				s.logger.Error("failed to semantic cache chat response", "error", err)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	if routingDecision != nil {
		w.Header().Set("X-Routing-Class", string(routingDecision.Class))
		w.Header().Set("X-Routing-Model", routingDecision.SelectedModel)
		w.Header().Set("X-Routing-Scores", fmt.Sprintf("FAST=%.2f,BALANCED=%.2f,DEEP=%.2f",
			routingDecision.FinalScores[router.RouteFast],
			routingDecision.FinalScores[router.RouteBalanced],
			routingDecision.FinalScores[router.RouteDeep]))
	}
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		s.logger.Error("failed to encode chat response", "error", err)
	}
}

func (s *Server) handleChatStream(w http.ResponseWriter, r *http.Request, req *llm.ChatRequest) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	if s.modelRouter != nil {
		prompt := extractPrompt(req.Messages)
		if prompt != "" {
			dec, err := s.modelRouter.Route(r.Context(), prompt, nil)
			if err == nil && dec != nil {
				req.Model = dec.SelectedModel
				w.Header().Set("X-Routing-Class", string(dec.Class))
				w.Header().Set("X-Routing-Model", dec.SelectedModel)
				w.Header().Set("X-Routing-Scores", fmt.Sprintf("FAST=%.2f,BALANCED=%.2f,DEEP=%.2f",
					dec.FinalScores[router.RouteFast],
					dec.FinalScores[router.RouteBalanced],
					dec.FinalScores[router.RouteDeep]))
			}
		}
	}

	stream, err := s.provider.ChatStream(r.Context(), req)
	if err != nil {
		s.logger.Error("provider stream failed", "error", err)
		http.Error(w, fmt.Sprintf("provider error: %v", err), http.StatusBadGateway)
		return
	}
	defer stream.Close()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Fprintf(w, "%s\n", line)
		flusher.Flush()
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		s.logger.Error("error reading from stream", "error", err)
	}
}

func extractPrompt(messages []llm.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			return messages[i].Content
		}
	}
	if len(messages) > 0 {
		return messages[len(messages)-1].Content
	}
	return ""
}
