package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"
)

func (s Server) handleFetchURLs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	urls, err := s.validateUrls(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	res, err := fetchURLs(r.Context(), int(s.config.MaxOutcomeRequests), s.config.MaxRequestTimeout, urls)
	if err != nil {
		http.Error(w, fmt.Sprintf("url fetching failed: %s", err), http.StatusInternalServerError)
		return
	}

	resp := make(map[string]string, len(res))
	for _, r := range res {
		resp[r.URL] = r.Result
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("error encode response: %s", err)
	}
}

func fetchURLs(ctx context.Context, maxOutcomeRequests int, reqTimeout time.Duration, urls []string) (res []Data, firstErr error) {
	errorCh := make(chan error)
	errorProcessingDone := make(chan struct{})
	urlsCh := make(chan string, maxOutcomeRequests)
	go func() {
		defer close(urlsCh)

		for _, u := range urls {
			select {
			case <-ctx.Done():
				return
			case urlsCh <- u:
			case <-errorProcessingDone:
				return
			}
		}
	}()

	ctx, cancel := context.WithCancel(ctx)

	go func() {
		for err := range errorCh {
			select {
			case <-ctx.Done():
			default:
				firstErr = err // return the first error
			}
			cancel()
		}
		close(errorProcessingDone)
	}()

	var (
		workResults Results
		wg          sync.WaitGroup
	)
	for i := 0; i < maxOutcomeRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := startFetchURLWorker(ctx, reqTimeout, urlsCh)
			if err != nil {
				errorCh <- err
				return
			}
			workResults.Append(res...)
		}()
	}
	wg.Wait()
	close(errorCh)

	<-errorProcessingDone

	if firstErr != nil {
		return nil, firstErr
	}
	return workResults.Get(), nil
}

func (s Server) validateUrls(r *http.Request) ([]string, error) {
	var URLs []string
	if err := json.NewDecoder(r.Body).Decode(&URLs); err != nil {
		return nil, fmt.Errorf("couldn't decode request: %s", err)
	}
	if len(URLs) > int(s.config.MaxCountUrls) {
		return nil, fmt.Errorf("max number of urls is %d", s.config.MaxCountUrls)
	}
	for _, rawURL := range URLs {
		// if _, err := url.Parse(rawURL); err != nil { // not working??
		if _, err := url.ParseRequestURI(rawURL); err != nil && !isUrl(rawURL) {
			return nil, fmt.Errorf("url %q is invalid", rawURL)
		}
	}

	return URLs, nil
}

func startFetchURLWorker(ctx context.Context, reqTimeout time.Duration, urlsCh <-chan string) (res []Data, err error) {
	for {
		select {
		case <-ctx.Done():
			return nil, nil
		case u, ok := <-urlsCh:
			if !ok {
				return res, nil
			}

			result, err := fetchURL(ctx, reqTimeout, u)
			if err != nil {
				return nil, fmt.Errorf("couldn't fetch %q: %w", u, err)
			}
			res = append(res, Data{
				URL:    u,
				Result: result,
			})
		}
	}
}

func fetchURL(ctx context.Context, reqTimeout time.Duration, url string) (body string, err error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("couldn't build request: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, reqTimeout)
	defer cancel()

	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("couldn't read body: %w", err)
	}
	return string(b), nil
}

type Data struct {
	URL    string
	Result string
}

type Results struct {
	mu  sync.Mutex
	res []Data
}

func (res *Results) Append(s ...Data) {
	res.mu.Lock()
	defer res.mu.Unlock()

	res.res = append(res.res, s...)
}

func (res *Results) Get() []Data {
	// if read only one thread
	// res.mu.Lock()
	// defer res.mu.Unlock()

	return res.res
}

// stackoverflow like!!
func isUrl(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}
