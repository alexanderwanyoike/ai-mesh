package provider

import (
	"context"
	"fmt"
)

func init() { Register(&Pixal3D{}) }

// Pixal3D implements Provider using TencentARC Pixal3D on Fal (pixel-aligned,
// high-fidelity image-to-3d). Image-to-3d only.
type Pixal3D struct{ falQueue }

func (p *Pixal3D) Name() string         { return "pixal3d" }
func (p *Pixal3D) DefaultModel() string { return "fal-ai/pixal3d" }
func (p *Pixal3D) APIKeyEnv() string    { return "FAL_API_KEY" }
func (p *Pixal3D) APIKeyURL() string    { return "https://fal.ai/dashboard/keys" }

func (p *Pixal3D) payload(req *GenerateRequest) (map[string]any, error) {
	if req.APIKey == "" {
		return nil, fmt.Errorf("no API key provided for pixal3d")
	}
	if req.InputImage == nil {
		return nil, fmt.Errorf("pixal3d is image-to-3d only: pass an image with -i, or pipe one from ai-img")
	}
	payload := map[string]any{
		"image_url": dataURI(req.InputImage, req.InputMIME),
	}
	// Pixal3D controls poly count via decimation_target, which only applies when
	// remesh is on. --faces maps to it; --pbr has no effect (Pixal3D always
	// textures).
	if req.FaceCount > 0 {
		payload["remesh"] = true
		payload["decimation_target"] = req.FaceCount
	}
	return payload, nil
}

func (p *Pixal3D) Submit(ctx context.Context, req *GenerateRequest) (string, error) {
	payload, err := p.payload(req)
	if err != nil {
		return "", err
	}
	s, err := p.submit(ctx, req.APIKey, modelOr(req.Model, p.DefaultModel()), payload)
	if err != nil {
		return "", err
	}
	return s.RequestID, nil
}

func (p *Pixal3D) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	payload, err := p.payload(req)
	if err != nil {
		return nil, err
	}
	return p.run(ctx, req.APIKey, modelOr(req.Model, p.DefaultModel()), payload)
}

func (p *Pixal3D) Fetch(ctx context.Context, apiKey, requestID string) (*GenerateResponse, error) {
	return p.fetch(ctx, apiKey, "fal-ai/pixal3d", requestID)
}
