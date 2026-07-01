package provider

import (
	"context"
	"strings"
	"testing"
)

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

func TestPixal3DRejectsTextOnly(t *testing.T) {
	_, err := (&Pixal3D{}).Generate(context.Background(), &GenerateRequest{APIKey: "k", Prompt: "a dragon"})
	if err == nil || !strings.Contains(err.Error(), "image-to-3d only") {
		t.Errorf("expected image-to-3d-only error, got %v", err)
	}
}

func TestPixal3DRequiresKey(t *testing.T) {
	_, err := (&Pixal3D{}).Generate(context.Background(), &GenerateRequest{InputImage: []byte("x")})
	if err == nil || !strings.Contains(err.Error(), "no API key") {
		t.Errorf("expected missing-key error, got %v", err)
	}
}
