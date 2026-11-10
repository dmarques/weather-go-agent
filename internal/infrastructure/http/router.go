package http

import stdhttp "net/http"

// NewRouter creates a new HTTP router with the provided handler, registering the health check and chat endpoints.
func NewRouter(handler *Handler) *stdhttp.ServeMux {
	mux := stdhttp.NewServeMux()
	mux.HandleFunc("GET /healthz", handler.Health)
	mux.HandleFunc("POST /chat", handler.Chat)
	return mux
}
