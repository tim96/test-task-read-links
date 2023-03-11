package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tim96/test-task-read-links/internal"
)

func main() {
	var (
		port               = flag.Uint64("port", 8080, "http port")
		maxCountUrls       = flag.Uint64("max_count_urls", 20, "max available count urls for parsing")
		maxRequests        = flag.Uint64("max_requests", 100, "max available requests")
		maxOutcomeRequests = flag.Uint64("max_outcome_requests", 4, "max available outcome requests")
		maxOutcomeTimeout  = flag.Uint64("max_outcome_timeout", 1, "max available outcome timeout(seconds)")
	)
	flag.Parse()

	config := internal.Config{
		Port:                  int(*port),
		MaxCountUrls:          *maxCountUrls,
		MaxConcurrentRequests: *maxRequests,
		MaxOutcomeRequests:    *maxOutcomeRequests,
		MaxRequestTimeout:     time.Second * time.Duration(*maxOutcomeTimeout),
	}
	s := internal.NewServer(config)

	errCh := make(chan error, 1)
	go func() {
		if err := s.Start(); err != nil {
			errCh <- err
		}
	}()

	termCh := make(chan os.Signal, 1)
	signal.Notify(termCh, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-termCh:
		log.Println("interrupt signal")
	case err := <-errCh:
		log.Printf("server error, stop app: %s", err)
	}

	log.Println("stop server")
	if err := s.Stop(); err != nil {
		log.Printf("error shutdown server: %s", err)
	}
}
