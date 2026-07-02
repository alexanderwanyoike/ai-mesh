package provider

import (
	"reflect"
	"testing"
)

func TestRegistry(t *testing.T) {
	// Providers are hosts (fal, meshy); models (hunyuan3d, tripo, ...) are chosen
	// per request, not registered as top-level providers.
	want := []string{"fal", "meshy"}
	if got := List(); !reflect.DeepEqual(got, want) {
		t.Errorf("List() = %v, want %v", got, want)
	}
	for _, name := range want {
		if Get(name) == nil {
			t.Errorf("Get(%q) = nil", name)
		}
	}
	if Get("nope") != nil {
		t.Error("Get(nope) should be nil")
	}
}

func TestFalModels(t *testing.T) {
	want := []string{"hunyuan3d", "hunyuan3d-rapid", "pixal3d", "tripo", "rodin"}
	if got := (&Fal{}).Models(); !reflect.DeepEqual(got, want) {
		t.Errorf("Fal.Models() = %v, want %v", got, want)
	}
	if def := (&Fal{}).DefaultModel(); def != "hunyuan3d" {
		t.Errorf("Fal.DefaultModel() = %q, want hunyuan3d", def)
	}
}

func TestDetectMIMEType(t *testing.T) {
	cases := map[string]string{
		"a.png":  "image/png",
		"a.jpg":  "image/jpeg",
		"a.jpeg": "image/jpeg",
		"a.webp": "image/webp",
		"a.gif":  "image/gif",
		"a.bmp":  "image/png", // default
	}
	for path, want := range cases {
		if got := DetectMIMEType(path); got != want {
			t.Errorf("DetectMIMEType(%q) = %q, want %q", path, got, want)
		}
	}
}
