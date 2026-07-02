package provider

import (
	"context"
	"strings"
	"testing"
)

// -m rodin routes to the Rodin endpoint; --pbr selects the PBR material.
func TestFalRodinModel(t *testing.T) {
	srv, auth, payload := falMoreServer(t, "/fal-ai/hyper3d/rodin", "model_mesh")
	defer srv.Close()

	f := &Fal{falQueue{HTTPClient: srv.Client(), BaseURL: srv.URL}}
	resp, err := f.Generate(context.Background(), &GenerateRequest{
		APIKey:     "test-key",
		Model:      "rodin",
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

func TestFalRodinDefaultsToShadedWithoutPBR(t *testing.T) {
	srv, _, payload := falMoreServer(t, "/fal-ai/hyper3d/rodin", "model_mesh")
	defer srv.Close()

	f := &Fal{falQueue{HTTPClient: srv.Client(), BaseURL: srv.URL}}
	if _, err := f.Generate(context.Background(), &GenerateRequest{
		APIKey:     "k",
		Model:      "rodin",
		InputImage: []byte("img"),
		InputMIME:  "image/png",
	}); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if (*payload)["material"] != "Shaded" {
		t.Errorf("material = %v, want Shaded (no --pbr)", (*payload)["material"])
	}
}
