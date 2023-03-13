package internal

import (
	"net/http"
	"sync"
)

func rateLimit(h http.Handler, maxConcurrencyRequests uint64) http.Handler {
	var reqCount uint64
	var mu sync.RWMutex

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		if reqCount >= maxConcurrencyRequests {
			mu.Unlock()
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}
		reqCount++
		mu.Unlock()

		defer func() {
			mu.Lock()
			reqCount--
			mu.Unlock()
		}()

		h.ServeHTTP(w, r)
	})
}
