package internal

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_handler(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		expCode int
		expBody string
	}{
		{
			name:    "max count urls",
			body:    `["1","2","3"]`,
			expCode: http.StatusBadRequest,
		},
		{
			name:    "bad url",
			body:    `["/1"]`,
			expCode: http.StatusInternalServerError,
		},
		{
			name:    "bad url",
			body:    `["sss"]`,
			expCode: http.StatusBadRequest,
		},
		{
			name:    "response ok",
			body:    `["https://google.com"]`,
			expCode: http.StatusOK,
		},
		{
			name: "response ok with 404",
			// replace by my own server
			body:    `["https://google.com/12"]`,
			expCode: http.StatusOK,
		},
	}

	server := NewServer(Config{
		MaxConcurrentRequests: 100,
		MaxRequestTimeout:     time.Second * time.Duration(2),
		MaxOutcomeRequests:    4,
		MaxCountUrls:          2,
		Port:                  8080,
	})

	for _, test := range tests {
		tt := test
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/", strings.NewReader(tt.body))
			req.Header.Add("Content-Type", "application/json")

			server.handleFetchURLs(w, req)

			if w.Code != tt.expCode {
				t.Error(w.Body.String())
			}

			if tt.expCode == http.StatusOK && len(tt.expBody) > 0 {
				assert.JSONEq(t, tt.expBody, w.Body.String())
			}
		})
	}
}
