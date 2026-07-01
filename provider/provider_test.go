package provider

import (
	"reflect"
	"testing"
)

func TestRegistry(t *testing.T) {
	want := []string{"fal", "meshy", "pixal3d", "rodin", "tripo"}
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
