package provider

import (
	"context"
	"path/filepath"
	"sort"
	"strings"
)

// GenerateRequest holds the parameters for a mesh generation request.
type GenerateRequest struct {
	APIKey     string // resolved by the caller (flag > env > config)
	Prompt     string // text prompt (text-to-3d); ignored when InputImage is set
	Model      string // provider-specific model/endpoint; empty uses the default
	InputImage []byte // reference image for image-to-3d; nil for text-to-3d
	InputMIME  string // e.g. "image/png"
	FaceCount  int    // target polygon count
	PBR        bool   // request PBR material maps
}

// GenerateResponse holds the result of a mesh generation request.
type GenerateResponse struct {
	ModelData []byte // the downloaded model file (GLB)
	Format    string // "glb"
	URL       string // hosted URL the model was downloaded from
	Message   string // optional human-readable note from the provider
}

// Provider defines the interface a hosting backend must implement. A provider is
// the host/API you authenticate against (Fal, Meshy); the generator you run is
// the model, selected per request via GenerateRequest.Model and enumerated by
// Models (e.g. Fal hosts hunyuan3d, pixal3d, tripo, rodin).
type Provider interface {
	Name() string
	// DefaultModel is the model used when a request names none.
	DefaultModel() string
	// Models lists the model names this provider offers, for help and validation.
	Models() []string
	// APIKeyEnv is the environment variable this provider's key is read from.
	APIKeyEnv() string
	// APIKeyURL is where a user can obtain an API key.
	APIKeyURL() string
	// Generate submits a job and waits for the finished model.
	Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)
	// Submit fires a job and returns a provider job ID without waiting, for
	// fire-and-forget use. Some inputs may not support it (returns an error).
	Submit(ctx context.Context, req *GenerateRequest) (jobID string, err error)
	// Fetch waits for an already-submitted job (by ID) and returns the model. The
	// model name is needed to route the request (e.g. Fal's per-model app base).
	Fetch(ctx context.Context, apiKey, model, jobID string) (*GenerateResponse, error)
}

var registry = map[string]Provider{}

// Register adds a provider to the registry.
func Register(p Provider) {
	registry[p.Name()] = p
}

// Get returns a provider by name, or nil if not found.
func Get(name string) Provider {
	return registry[name]
}

// List returns the names of all registered providers, sorted.
func List() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// DetectMIMEType returns the MIME type for an image file based on its extension.
func DetectMIMEType(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return "image/png"
	}
}
