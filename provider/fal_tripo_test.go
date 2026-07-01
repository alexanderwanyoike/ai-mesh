package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// -m tripo routes to the Tripo endpoint; Tripo also returns its GLB under
// model_mesh.url, exercising the falGLBURL fallback.
func TestFalTripoModel(t *testing.T) {
	srv, auth, payload := falMoreServer(t, "/tripo3d/tripo/v2.5/image-to-3d", "model_mesh")
	defer srv.Close()

	f := &Fal{falQueue{HTTPClient: srv.Client(), BaseURL: srv.URL}}
	resp, err := f.Generate(context.Background(), &GenerateRequest{
		APIKey:     "test-key",
		Model:      "tripo",
		InputImage: []byte("imagebytes"),
		InputMIME:  "image/png",
		FaceCount:  40000,
		PBR:        true,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if string(resp.ModelData) != "GLB_BYTES" {
		t.Errorf("ModelData = %q, want GLB_BYTES", resp.ModelData)
	}
	if *auth != "Key test-key" {
		t.Errorf("Authorization = %q, want Key test-key", *auth)
	}
	if (*payload)["pbr"] != true {
		t.Errorf("pbr = %v, want true", (*payload)["pbr"])
	}
	if (*payload)["face_limit"].(float64) != 40000 {
		t.Errorf("face_limit = %v, want 40000", (*payload)["face_limit"])
	}
}

// Fetch-by-id derives the app base from the model's endpoint; for tripo that is
// tripo3d/tripo (not the full versioned model path).
func TestFalTripoFetchByID(t *testing.T) {
	pollInterval = time.Millisecond

	var srvURL string
	mux := http.NewServeMux()
	mux.HandleFunc("/tripo3d/tripo/requests/job-9/status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"status": "COMPLETED"})
	})
	mux.HandleFunc("/tripo3d/tripo/requests/job-9", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"model_mesh": map[string]any{"url": srvURL + "/m.glb"}})
	})
	mux.HandleFunc("/m.glb", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("FETCHED")) })

	srv := httptest.NewServer(mux)
	defer srv.Close()
	srvURL = srv.URL

	f := &Fal{falQueue{HTTPClient: srv.Client(), BaseURL: srv.URL}}
	resp, err := f.Fetch(context.Background(), "k", "tripo", "job-9")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if string(resp.ModelData) != "FETCHED" {
		t.Errorf("ModelData = %q, want FETCHED", resp.ModelData)
	}
}
