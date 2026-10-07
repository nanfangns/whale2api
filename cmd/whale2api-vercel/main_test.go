package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVercelHandlerRoutesPoolUIAndGateway(t *testing.T) {
	api := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	admin := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})
	h := vercelHandler{api: api, admin: admin}

	for _, tc := range []struct {
		path string
		want int
	}{
		{path: "/", want: http.StatusAccepted},
		{path: "/app.js", want: http.StatusAccepted},
		{path: "/api/keys", want: http.StatusAccepted},
		{path: "/v1/chat/completions", want: http.StatusNoContent},
		{path: "/healthz", want: http.StatusNoContent},
	} {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			res := httptest.NewRecorder()
			h.ServeHTTP(res, req)
			if res.Code != tc.want {
				t.Fatalf("status = %d, want %d", res.Code, tc.want)
			}
		})
	}
}
