package main

import (
	"log"
	"net/http"
	"time"

	"github.com/dmarques/weather-go-agent/internal/application/chat"
	"github.com/dmarques/weather-go-agent/internal/config"
	"github.com/dmarques/weather-go-agent/internal/infrastructure/external"
	httpinfra "github.com/dmarques/weather-go-agent/internal/infrastructure/http"
	"github.com/dmarques/weather-go-agent/pkg/constant"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Set up HTTP clients and services
	httpClient := &http.Client{Timeout: constant.RequestTimeout}
	openAI := external.NewOpenAIClient(httpClient, cfg.OpenAIKey)
	openMeteo := external.NewOpenMeteoClient(httpClient)
	chatService := chat.NewService(cfg.OpenAIModel, openAI, openMeteo)

	// Set up HTTP server and routes
	handler := httpinfra.NewHandler(
		chatService,
		cfg.AgentToken,
		httpinfra.NewRateLimiter(),
		make(chan struct{}, constant.MaxConcurrentRequests),
	)
	mux := httpinfra.NewRouter(handler)

	// Start the HTTP server
	server := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           httpinfra.SecurityHeaders(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    8 * 1024,
	}

	// Log the listening address and start the server
	log.Printf("weather agent listening on %s", cfg.ListenAddr)
	log.Fatal(server.ListenAndServe())
}
