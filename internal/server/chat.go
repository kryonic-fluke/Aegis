package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/kryonic-fluke/aegis/internal/llm"
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
	resp, err := s.provider.Chat(r.Context(), req)
	if err != nil {
		s.logger.Error("provider chat failed", "error", err)
		http.Error(w, fmt.Sprintf("provider error: %v", err), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
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
