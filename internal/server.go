package internal

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

type Config struct {
	Port                  int
	MaxCountUrls          uint64
	MaxOutcomeRequests    uint64
	MaxConcurrentRequests uint64
	MaxRequestTimeout     time.Duration
}

type Server struct {
	config Config
	http   *http.Server
}

func NewServer(config Config) *Server {
	s := &Server{
		config: config,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleFetchURLs)

	handler := rateLimit(mux, config.MaxConcurrentRequests)

	s.http = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.config.Port),
		Handler: handler,
	}

	return s
}

func (s Server) Start() error {
	if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return s.http.Shutdown(ctx)
}
