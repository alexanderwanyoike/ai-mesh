package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFalGenerateImageToMesh(t *testing.T) {
	pollInterval = time.Millisecond

	var srvURL string
	var gotPayload map[string]any
	var gotAuth string

	mux := http.NewServeMux()
	mux.HandleFunc("/fal-ai/hunyuan-3d/v3.1/pro/image-to-3d", func(w http.ResponseWriter, r *http.Request) {
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
		writeJSON(w, map[string]any{
			"model_glb": map[string]any{"url": srvURL + "/model.glb"},
			"seed":      7,
		})
	})
	mux.HandleFunc("/model.glb", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("GLB_BYTES"))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	srvURL = srv.URL

	f := &Fal{falQueue{HTTPClient: srv.Client(), BaseURL: srv.URL}}
	resp, err := f.Generate(context.Background(), &GenerateRequest{
		APIKey:     "test-key",
		InputImage: []byte("imagebytes"),
		InputMIME:  "image/png",
		FaceCount:  100000,
		PBR:        true,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if string(resp.ModelData) != "GLB_BYTES" {
		t.Errorf("ModelData = %q, want GLB_BYTES", resp.ModelData)
	}
	if resp.URL != srvURL+"/model.glb" {
		t.Errorf("URL = %q", resp.URL)
	}
	if gotAuth != "Key test-key" {
		t.Errorf("Authorization = %q, want Key test-key", gotAuth)
	}
	img, _ := gotPayload["input_image_url"].(string)
	if !strings.HasPrefix(img, "data:image/png;base64,") {
		t.Errorf("input_image_url = %q, want data URI", img)
	}
	if gotPayload["face_count"].(float64) != 100000 {
		t.Errorf("face_count = %v, want 100000", gotPayload["face_count"])
	}
	if gotPayload["enable_pbr"].(bool) != true {
		t.Errorf("enable_pbr = %v, want true", gotPayload["enable_pbr"])
	}
}

func TestFalRejectsTextOnly(t *testing.T) {
	f := &Fal{}
	_, err := f.Generate(context.Background(), &GenerateRequest{APIKey: "test-key", Prompt: "a dragon"})
	if err == nil || !strings.Contains(err.Error(), "image-to-3d only") {
		t.Errorf("expected image-to-3d-only error, got %v", err)
	}
}

func TestFalRequiresKey(t *testing.T) {
	f := &Fal{}
	_, err := f.Generate(context.Background(), &GenerateRequest{InputImage: []byte("x")})
	if err == nil || !strings.Contains(err.Error(), "no API key") {
		t.Errorf("expected missing-key error, got %v", err)
	}
}

// model_glb is polymorphic (can be OBJ on some tiers); model_urls.glb is always
// GLB, so it must win when both are present.
func TestFalPrefersDedicatedGLBURL(t *testing.T) {
	pollInterval = time.Millisecond

	var srvURL string
	mux := http.NewServeMux()
	mux.HandleFunc("/fal-ai/hunyuan-3d/v3.1/rapid/image-to-3d", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"request_id": "req-2", "status_url": srvURL + "/status", "response_url": srvURL + "/response"})
	})
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"status": "COMPLETED"})
	})
	mux.HandleFunc("/response", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"model_glb":  map[string]any{"url": srvURL + "/model.obj"}, // wrong/polymorphic
			"model_urls": map[string]any{"glb": map[string]any{"url": srvURL + "/model.glb"}},
		})
	})
	mux.HandleFunc("/model.glb", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("GLB")) })
	mux.HandleFunc("/model.obj", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("OBJ")) })

	srv := httptest.NewServer(mux)
	defer srv.Close()
	srvURL = srv.URL

	f := &Fal{falQueue{HTTPClient: srv.Client(), BaseURL: srv.URL}}
	resp, err := f.Generate(context.Background(), &GenerateRequest{
		APIKey:     "k",
		Model:      "fal-ai/hunyuan-3d/v3.1/rapid/image-to-3d",
		InputImage: []byte("img"),
		InputMIME:  "image/png",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if resp.URL != srvURL+"/model.glb" || string(resp.ModelData) != "GLB" {
		t.Errorf("expected dedicated GLB url, got url=%q data=%q", resp.URL, resp.ModelData)
	}
}

func TestFalSubmitReturnsID(t *testing.T) {
	var srvURL string
	mux := http.NewServeMux()
	mux.HandleFunc("/fal-ai/hunyuan-3d/v3.1/pro/image-to-3d", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"request_id": "abc-123", "status_url": srvURL + "/s", "response_url": srvURL + "/r"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	srvURL = srv.URL

	f := &Fal{falQueue{HTTPClient: srv.Client(), BaseURL: srv.URL}}
	id, err := f.Submit(context.Background(), &GenerateRequest{APIKey: "k", InputImage: []byte("x"), InputMIME: "image/png"})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if id != "abc-123" {
		t.Errorf("id = %q, want abc-123", id)
	}
}

// Fetch must reconstruct the per-request URL from the app-base path, not the
// full versioned model path (which 405s on Fal).
func TestFalFetchByID(t *testing.T) {
	pollInterval = time.Millisecond

	var srvURL string
	mux := http.NewServeMux()
	mux.HandleFunc("/fal-ai/hunyuan-3d/requests/job-9/status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"status": "COMPLETED"})
	})
	mux.HandleFunc("/fal-ai/hunyuan-3d/requests/job-9", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"model_urls": map[string]any{"glb": map[string]any{"url": srvURL + "/m.glb"}}})
	})
	mux.HandleFunc("/m.glb", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("FETCHED")) })

	srv := httptest.NewServer(mux)
	defer srv.Close()
	srvURL = srv.URL

	f := &Fal{falQueue{HTTPClient: srv.Client(), BaseURL: srv.URL}}
	resp, err := f.Fetch(context.Background(), "k", "job-9")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if string(resp.ModelData) != "FETCHED" {
		t.Errorf("ModelData = %q, want FETCHED", resp.ModelData)
	}
}
