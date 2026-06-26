package provider

import (
	"reflect"
	"testing"
)

func TestRegistry(t *testing.T) {
	if got := List(); !reflect.DeepEqual(got, []string{"fal", "meshy"}) {
		t.Errorf("List() = %v, want [fal meshy]", got)
	}
	if Get("fal") == nil {
		t.Error("Get(fal) = nil")
	}
	if Get("meshy") == nil {
		t.Error("Get(meshy) = nil")
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
