package provider

import (
	"context"
	"strings"
	"testing"
)

// -m pixal3d routes the Fal provider to the Pixal3D endpoint and payload.
func TestFalPixal3DModel(t *testing.T) {
	srv, auth, payload := falMoreServer(t, "/fal-ai/pixal3d", "model_glb")
	defer srv.Close()

	f := &Fal{falQueue{HTTPClient: srv.Client(), BaseURL: srv.URL}}
	resp, err := f.Generate(context.Background(), &GenerateRequest{
		APIKey:     "test-key",
		Model:      "pixal3d",
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
