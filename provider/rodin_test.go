package provider

import (
	"context"
	"strings"
	"testing"
)

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

func TestRodinRejectsTextOnly(t *testing.T) {
	_, err := (&Rodin{}).Generate(context.Background(), &GenerateRequest{APIKey: "k", Prompt: "a dragon"})
	if err == nil || !strings.Contains(err.Error(), "image-to-3d only") {
		t.Errorf("expected image-to-3d-only error, got %v", err)
	}
}

func TestRodinRequiresKey(t *testing.T) {
	_, err := (&Rodin{}).Generate(context.Background(), &GenerateRequest{InputImage: []byte("x")})
	if err == nil || !strings.Contains(err.Error(), "no API key") {
		t.Errorf("expected missing-key error, got %v", err)
	}
}
