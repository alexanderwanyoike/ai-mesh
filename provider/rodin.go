package provider

import (
	"context"
	"fmt"
)

func init() { Register(&Rodin{}) }

// Rodin implements Provider using Hyper3D Rodin on Fal. Image-to-3d only.
type Rodin struct{ falQueue }

func (r *Rodin) Name() string         { return "rodin" }
func (r *Rodin) DefaultModel() string { return "fal-ai/hyper3d/rodin" }
func (r *Rodin) APIKeyEnv() string    { return "FAL_API_KEY" }
func (r *Rodin) APIKeyURL() string    { return "https://fal.ai/dashboard/keys" }

func (r *Rodin) payload(req *GenerateRequest) (map[string]any, error) {
	if req.APIKey == "" {
		return nil, fmt.Errorf("no API key provided for rodin")
	}
	if req.InputImage == nil {
		return nil, fmt.Errorf("rodin is image-to-3d only: pass an image with -i, or pipe one from ai-img")
	}
	// Rodin takes an array of image urls (multi-view); a single reference is the
	// common case here. --pbr selects PBR vs Shaded material; Rodin sets density
	// via its own quality tier, so --faces has no effect.
	material := "Shaded"
	if req.PBR {
		material = "PBR"
	}
	return map[string]any{
		"input_image_urls":     []string{dataURI(req.InputImage, req.InputMIME)},
		"geometry_file_format": "glb",
		"material":             material,
	}, nil
}

func (r *Rodin) Submit(ctx context.Context, req *GenerateRequest) (string, error) {
	payload, err := r.payload(req)
	if err != nil {
		return "", err
	}
	s, err := r.submit(ctx, req.APIKey, modelOr(req.Model, r.DefaultModel()), payload)
	if err != nil {
		return "", err
	}
	return s.RequestID, nil
}

func (r *Rodin) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	payload, err := r.payload(req)
	if err != nil {
		return nil, err
	}
	return r.run(ctx, req.APIKey, modelOr(req.Model, r.DefaultModel()), payload)
}

func (r *Rodin) Fetch(ctx context.Context, apiKey, requestID string) (*GenerateResponse, error) {
	return r.fetch(ctx, apiKey, "fal-ai/hyper3d", requestID)
}
