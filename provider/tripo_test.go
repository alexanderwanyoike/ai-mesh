package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTripoGenerateUsesModelMesh(t *testing.T) {
	// Tripo returns its GLB under model_mesh.url, so this also exercises the
	// falGLBURL fallback to model_mesh.
	srv, auth, payload := falMoreServer(t, "/tripo3d/tripo/v2.5/image-to-3d", "model_mesh")
	defer srv.Close()

	tr := &Tripo{falQueue{HTTPClient: srv.Client(), BaseURL: srv.URL}}
	resp, err := tr.Generate(context.Background(), &GenerateRequest{
		APIKey:     "test-key",
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

func TestTripoRejectsTextOnly(t *testing.T) {
	_, err := (&Tripo{}).Generate(context.Background(), &GenerateRequest{APIKey: "k", Prompt: "a dragon"})
	if err == nil || !strings.Contains(err.Error(), "image-to-3d only") {
		t.Errorf("expected image-to-3d-only error, got %v", err)
	}
}

func TestTripoRequiresKey(t *testing.T) {
	_, err := (&Tripo{}).Generate(context.Background(), &GenerateRequest{InputImage: []byte("x")})
	if err == nil || !strings.Contains(err.Error(), "no API key") {
		t.Errorf("expected missing-key error, got %v", err)
	}
}

func TestTripoFetchByID(t *testing.T) {
	pollInterval = time.Millisecond

	var srvURL string
	mux := http.NewServeMux()
	// Fetch reconstructs .../{appBase}/requests/{id}; for Tripo the app base is
	// tripo3d/tripo (not the full versioned model path).
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

	tr := &Tripo{falQueue{HTTPClient: srv.Client(), BaseURL: srv.URL}}
	resp, err := tr.Fetch(context.Background(), "k", "job-9")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if string(resp.ModelData) != "FETCHED" {
		t.Errorf("ModelData = %q, want FETCHED", resp.ModelData)
	}
}
