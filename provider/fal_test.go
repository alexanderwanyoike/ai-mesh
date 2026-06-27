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

	f := &Fal{HTTPClient: srv.Client(), BaseURL: srv.URL}
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

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
