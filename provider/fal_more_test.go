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

func TestPixal3DGenerate(t *testing.T) {
	srv, auth, payload := falMoreServer(t, "/fal-ai/pixal3d", "model_glb")
	defer srv.Close()

	p := &Pixal3D{falQueue{HTTPClient: srv.Client(), BaseURL: srv.URL}}
	resp, err := p.Generate(context.Background(), &GenerateRequest{
		APIKey:     "test-key",
		InputImage: []byte("imagebytes"),
		InputMIME:  "image/png",
		FaceCount:  100000,
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
	if img, _ := (*payload)["image_url"].(string); !strings.HasPrefix(img, "data:image/png;base64,") {
		t.Errorf("image_url = %q, want data URI", img)
	}
	if (*payload)["remesh"] != true {
		t.Errorf("remesh = %v, want true", (*payload)["remesh"])
	}
	if (*payload)["decimation_target"].(float64) != 100000 {
		t.Errorf("decimation_target = %v, want 100000", (*payload)["decimation_target"])
	}
}

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

func TestRodinGenerate(t *testing.T) {
	srv, auth, payload := falMoreServer(t, "/fal-ai/hyper3d/rodin", "model_mesh")
	defer srv.Close()

	r := &Rodin{falQueue{HTTPClient: srv.Client(), BaseURL: srv.URL}}
	resp, err := r.Generate(context.Background(), &GenerateRequest{
		APIKey:     "test-key",
		InputImage: []byte("imagebytes"),
		InputMIME:  "image/png",
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
	urls, ok := (*payload)["input_image_urls"].([]any)
	if !ok || len(urls) != 1 {
		t.Fatalf("input_image_urls = %v, want a 1-element array", (*payload)["input_image_urls"])
	}
	if img, _ := urls[0].(string); !strings.HasPrefix(img, "data:image/png;base64,") {
		t.Errorf("input_image_urls[0] = %q, want data URI", img)
	}
	if (*payload)["material"] != "PBR" {
		t.Errorf("material = %v, want PBR (--pbr set)", (*payload)["material"])
	}
	if (*payload)["geometry_file_format"] != "glb" {
		t.Errorf("geometry_file_format = %v, want glb", (*payload)["geometry_file_format"])
	}
}

func TestRodinDefaultsToShadedWithoutPBR(t *testing.T) {
	srv, _, payload := falMoreServer(t, "/fal-ai/hyper3d/rodin", "model_mesh")
	defer srv.Close()

	r := &Rodin{falQueue{HTTPClient: srv.Client(), BaseURL: srv.URL}}
	if _, err := r.Generate(context.Background(), &GenerateRequest{
		APIKey:     "k",
		InputImage: []byte("img"),
		InputMIME:  "image/png",
	}); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if (*payload)["material"] != "Shaded" {
		t.Errorf("material = %v, want Shaded (no --pbr)", (*payload)["material"])
	}
}

func TestMoreProvidersRejectTextOnly(t *testing.T) {
	for _, p := range []Provider{&Pixal3D{}, &Tripo{}, &Rodin{}} {
		_, err := p.Generate(context.Background(), &GenerateRequest{APIKey: "k", Prompt: "a dragon"})
		if err == nil || !strings.Contains(err.Error(), "image-to-3d only") {
			t.Errorf("%s: expected image-to-3d-only error, got %v", p.Name(), err)
		}
	}
}

func TestMoreProvidersRequireKey(t *testing.T) {
	for _, p := range []Provider{&Pixal3D{}, &Tripo{}, &Rodin{}} {
		_, err := p.Generate(context.Background(), &GenerateRequest{InputImage: []byte("x")})
		if err == nil || !strings.Contains(err.Error(), "no API key") {
			t.Errorf("%s: expected missing-key error, got %v", p.Name(), err)
		}
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
