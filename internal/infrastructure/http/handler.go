package http

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/dmarques/weather-go-agent/internal/application/chat"
	"github.com/dmarques/weather-go-agent/pkg/constant"
)

type ChatUseCase interface {
	Run(context.Context, string) (string, *chat.Usage, error)
}

type Handler struct {
	chatService ChatUseCase
	agentToken  string
	limiter     *RateLimiter
	semaphore   chan struct{}
}

// NewHandler creates a new Handler with the provided ChatUseCase, agent token, rate limiter,
// and semaphore for concurrency control.
func NewHandler(chatService ChatUseCase, agentToken string, limiter *RateLimiter, semaphore chan struct{}) *Handler {
	return &Handler{
		chatService: chatService,
		agentToken:  agentToken,
		limiter:     limiter,
		semaphore:   semaphore,
	}
}

type chatRequest struct {
	Message string `json:"message"`
}

type chatResponse struct {
	Answer string      `json:"answer,omitempty"`
	Usage  *chat.Usage `json:"usage,omitempty"`
	Error  string      `json:"error,omitempty"`
}

// Health handles the health check endpoint, returning a simple JSON response indicating the service is operational.
func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Chat handles the chat endpoint, processing incoming requests to generate responses based on the provided message.
// It performs authorization, rate limiting, and concurrency control before invoking the chat service.
func (h *Handler) Chat(w http.ResponseWriter, r *http.Request) {
	if !authorized(r, h.agentToken) {
		writeJSON(w, http.StatusUnauthorized, chatResponse{Error: "unauthorized"})
		return
	}

	if !h.limiter.Allow(clientIP(r)) {
		writeJSON(w, http.StatusTooManyRequests, chatResponse{Error: "rate limit exceeded"})
		return
	}

	select {
	case h.semaphore <- struct{}{}:
		defer func() { <-h.semaphore }()
	default:
		writeJSON(w, http.StatusServiceUnavailable, chatResponse{Error: "server busy"})
		return
	}

	var req chatRequest
	r.Body = http.MaxBytesReader(w, r.Body, constant.MaxMessageBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, chatResponse{Error: "invalid JSON body"})
		return
	}
	req.Message = strings.TrimSpace(req.Message)
	if req.Message == "" || len([]byte(req.Message)) > constant.MaxMessageBytes {
		writeJSON(w, http.StatusBadRequest, chatResponse{Error: "message is empty or too long"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constant.RequestTimeout)
	defer cancel()

	answer, usage, err := h.chatService.Run(ctx, req.Message)
	if err != nil {
		log.Printf("agent request failed: %v", err)
		writeJSON(w, http.StatusBadGateway, chatResponse{Error: "agent temporarily unavailable"})
		return
	}

	writeJSON(w, http.StatusOK, chatResponse{Answer: answer, Usage: usage})
}

// authorized checks if the request contains the correct agent token in the "X-Agent-Token" header,
// using constant-time comparison to prevent timing attacks.
func authorized(r *http.Request, expected string) bool {
	provided := r.Header.Get("X-Agent-Token")
	if provided == "" || len(provided) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}

// clientIP extracts the client's IP address from the request's RemoteAddr,
// handling cases where the address may include a port.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

// writeJSON writes the provided value as a JSON response with the specified HTTP status code,
// setting the appropriate content type header.
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
