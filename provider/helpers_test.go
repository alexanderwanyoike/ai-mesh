package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// falMoreServer stands up a queue submit -> status -> response -> download flow
// for a given model endpoint, capturing the auth header and request payload. The
// finished result nests the GLB url under glbField ("model_glb" or "model_mesh")
// so each provider's output shape can be exercised.
func falMoreServer(t *testing.T, endpoint, glbField string) (srv *httptest.Server, auth *string, payload *map[string]any) {
	t.Helper()
	pollInterval = time.Millisecond

	var gotAuth string
	var gotPayload map[string]any
	var srvURL string

	mux := http.NewServeMux()
	mux.HandleFunc(endpoint, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&gotPayload)
		writeJSON(w, map[string]any{
			"request_id":   "req-1",
			"status_url":   srvURL + "/status",
			"response_url": srvURL + "/response",
		})
	})
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"status": "COMPLETED"})
	})
	mux.HandleFunc("/response", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{glbField: map[string]any{"url": srvURL + "/model.glb"}})
	})
	mux.HandleFunc("/model.glb", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("GLB_BYTES"))
	})

	srv = httptest.NewServer(mux)
	srvURL = srv.URL
	return srv, &gotAuth, &gotPayload
}
