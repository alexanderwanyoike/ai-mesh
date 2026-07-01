package provider

import (
	"context"
	"fmt"
	"strings"
)

func init() { Register(&Fal{}) }

// falModel is one 3D model hosted on the Fal provider: the queue endpoint it
// submits to and how it builds its request payload (each model - Hunyuan3D,
// Pixal3D, Tripo, Rodin - takes different fields). The fetch-by-id app base is
// derived from the endpoint (see falAppBase).
type falModel struct {
	name     string
	endpoint string
	payload  func(req *GenerateRequest) (map[string]any, error)
}

// falModels is the set of models the Fal provider hosts, in display order. Fal is
// a hosting platform: it re-hosts Tencent's Hunyuan3D, TencentARC's Pixal3D,
// VAST's Tripo, and Deemos' Rodin, all behind one FAL_API_KEY. Select one with
// -m; the default is hunyuan3d.
var falModels = []falModel{
	{"hunyuan3d", "fal-ai/hunyuan-3d/v3.1/pro/image-to-3d", falHunyuanPayload},
	{"hunyuan3d-rapid", "fal-ai/hunyuan-3d/v3.1/rapid/image-to-3d", falHunyuanPayload},
	{"pixal3d", "fal-ai/pixal3d", falPixal3DPayload},
	{"tripo", "tripo3d/tripo/v2.5/image-to-3d", falTripoPayload},
	{"rodin", "fal-ai/hyper3d/rodin", falRodinPayload},
}

// Fal implements Provider as a hosting platform: one FAL_API_KEY, many models
// (image-to-3d only). The model is chosen per request via -m.
type Fal struct{ falQueue }

func (f *Fal) Name() string         { return "fal" }
func (f *Fal) DefaultModel() string { return "hunyuan3d" }
func (f *Fal) APIKeyEnv() string    { return "FAL_API_KEY" }
func (f *Fal) APIKeyURL() string    { return "https://fal.ai/dashboard/keys" }

func (f *Fal) Models() []string {
	names := make([]string, len(falModels))
	for i, m := range falModels {
		names[i] = m.name
	}
	return names
}

// resolveModel maps a -m value to a Fal model. Empty uses the default; a known
// friendly name (hunyuan3d, pixal3d, ...) maps to its endpoint; a value with a
// "/" is treated as a raw Fal endpoint id (advanced passthrough, Hunyuan-style
// payload).
func (f *Fal) resolveModel(name string) (falModel, error) {
	if name == "" {
		name = f.DefaultModel()
	}
	for _, m := range falModels {
		if m.name == name {
			return m, nil
		}
	}
	if strings.Contains(name, "/") {
		return falModel{name: name, endpoint: name, payload: falHunyuanPayload}, nil
	}
	return falModel{}, fmt.Errorf("unknown fal model %q (available: %s, or a raw fal-ai/... endpoint)",
		name, strings.Join(f.Models(), ", "))
}

// buildRequest validates the request and resolves the model + payload. The
// image-to-3d and key checks are shared across all Fal models.
func (f *Fal) buildRequest(req *GenerateRequest) (string, map[string]any, error) {
	if req.APIKey == "" {
		return "", nil, fmt.Errorf("no API key provided for fal")
	}
	if req.InputImage == nil {
		return "", nil, fmt.Errorf("fal is image-to-3d only: pass an image with -i, pipe one from ai-img, or use -p meshy for text-to-3d")
	}
	m, err := f.resolveModel(req.Model)
	if err != nil {
		return "", nil, err
	}
	payload, err := m.payload(req)
	if err != nil {
		return "", nil, err
	}
	return m.endpoint, payload, nil
}

func (f *Fal) Submit(ctx context.Context, req *GenerateRequest) (string, error) {
	endpoint, payload, err := f.buildRequest(req)
	if err != nil {
		return "", err
	}
	s, err := f.submit(ctx, req.APIKey, endpoint, payload)
	if err != nil {
		return "", err
	}
	return s.RequestID, nil
}

func (f *Fal) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	endpoint, payload, err := f.buildRequest(req)
	if err != nil {
		return nil, err
	}
	return f.run(ctx, req.APIKey, endpoint, payload)
}

func (f *Fal) Fetch(ctx context.Context, apiKey, model, requestID string) (*GenerateResponse, error) {
	m, err := f.resolveModel(model)
	if err != nil {
		return nil, err
	}
	return f.fetch(ctx, apiKey, falAppBase(m.endpoint), requestID)
}
