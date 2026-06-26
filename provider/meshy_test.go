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

func TestMeshyImageToMesh(t *testing.T) {
	pollInterval = time.Millisecond
	t.Setenv("MESHY_API_KEY", "test-key")

	var srvURL string
	var gotPayload map[string]any
	var gotAuth string

	mux := http.NewServeMux()
	mux.HandleFunc("/openapi/v1/image-to-3d", func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&gotPayload)
		writeJSON(w, map[string]any{"result": "img-1"})
	})
	mux.HandleFunc("/openapi/v1/image-to-3d/img-1", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"status":     "SUCCEEDED",
			"model_urls": map[string]any{"glb": srvURL + "/m.glb"},
		})
	})
	mux.HandleFunc("/m.glb", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("MESHY_GLB"))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	srvURL = srv.URL

	m := &Meshy{HTTPClient: srv.Client(), BaseURL: srv.URL}
	resp, err := m.Generate(context.Background(), &GenerateRequest{
		InputImage: []byte("imagebytes"),
		InputMIME:  "image/png",
		FaceCount:  30000,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if string(resp.ModelData) != "MESHY_GLB" {
		t.Errorf("ModelData = %q", resp.ModelData)
	}
	if gotAuth != "Bearer test-key" {
		t.Errorf("Authorization = %q, want Bearer test-key", gotAuth)
	}
	if img, _ := gotPayload["image_url"].(string); !strings.HasPrefix(img, "data:image/png;base64,") {
		t.Errorf("image_url = %q, want data URI", img)
	}
	if gotPayload["should_texture"].(bool) != true {
		t.Errorf("should_texture = %v, want true", gotPayload["should_texture"])
	}
	if gotPayload["target_polycount"].(float64) != 30000 {
		t.Errorf("target_polycount = %v", gotPayload["target_polycount"])
	}
}

func TestMeshyTextToMeshTwoStage(t *testing.T) {
	pollInterval = time.Millisecond
	t.Setenv("MESHY_API_KEY", "test-key")

	var srvURL string
	var modes []string

	mux := http.NewServeMux()
	mux.HandleFunc("/openapi/v2/text-to-3d", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		mode, _ := body["mode"].(string)
		modes = append(modes, mode)
		if mode == "preview" {
			writeJSON(w, map[string]any{"result": "prev-1"})
			return
		}
		// refine: must carry the preview task id
		if body["preview_task_id"] != "prev-1" {
			t.Errorf("refine preview_task_id = %v, want prev-1", body["preview_task_id"])
		}
		writeJSON(w, map[string]any{"result": "ref-1"})
	})
	mux.HandleFunc("/openapi/v2/text-to-3d/prev-1", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"status": "SUCCEEDED", "model_urls": map[string]any{}})
	})
	mux.HandleFunc("/openapi/v2/text-to-3d/ref-1", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"status":     "SUCCEEDED",
			"model_urls": map[string]any{"glb": srvURL + "/t.glb"},
		})
	})
	mux.HandleFunc("/t.glb", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("TEXT_GLB"))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	srvURL = srv.URL

	m := &Meshy{HTTPClient: srv.Client(), BaseURL: srv.URL}
	resp, err := m.Generate(context.Background(), &GenerateRequest{Prompt: "a low-poly chest"})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if string(resp.ModelData) != "TEXT_GLB" {
		t.Errorf("ModelData = %q, want TEXT_GLB", resp.ModelData)
	}
	if len(modes) != 2 || modes[0] != "preview" || modes[1] != "refine" {
		t.Errorf("modes = %v, want [preview refine]", modes)
	}
}

func TestMeshyTaskFailed(t *testing.T) {
	pollInterval = time.Millisecond
	t.Setenv("MESHY_API_KEY", "test-key")

	mux := http.NewServeMux()
	mux.HandleFunc("/openapi/v1/image-to-3d", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"result": "img-1"})
	})
	mux.HandleFunc("/openapi/v1/image-to-3d/img-1", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"status":     "FAILED",
			"task_error": map[string]any{"message": "bad image"},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	m := &Meshy{HTTPClient: srv.Client(), BaseURL: srv.URL}
	_, err := m.Generate(context.Background(), &GenerateRequest{InputImage: []byte("x"), InputMIME: "image/png"})
	if err == nil || !strings.Contains(err.Error(), "bad image") {
		t.Errorf("expected failed-task error, got %v", err)
	}
}

func TestMeshyRequiresKey(t *testing.T) {
	t.Setenv("MESHY_API_KEY", "")
	m := &Meshy{}
	_, err := m.Generate(context.Background(), &GenerateRequest{Prompt: "x"})
	if err == nil || !strings.Contains(err.Error(), "MESHY_API_KEY") {
		t.Errorf("expected MESHY_API_KEY error, got %v", err)
	}
}
