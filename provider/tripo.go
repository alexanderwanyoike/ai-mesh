package provider

import (
	"context"
	"fmt"
)

func init() { Register(&Tripo{}) }

// Tripo implements Provider using Tripo v2.5 on Fal. Image-to-3d only.
type Tripo struct{ falQueue }

func (t *Tripo) Name() string         { return "tripo" }
func (t *Tripo) DefaultModel() string { return "tripo3d/tripo/v2.5/image-to-3d" }
func (t *Tripo) APIKeyEnv() string    { return "FAL_API_KEY" }
func (t *Tripo) APIKeyURL() string    { return "https://fal.ai/dashboard/keys" }

func (t *Tripo) payload(req *GenerateRequest) (map[string]any, error) {
	if req.APIKey == "" {
		return nil, fmt.Errorf("no API key provided for tripo")
	}
	if req.InputImage == nil {
		return nil, fmt.Errorf("tripo is image-to-3d only: pass an image with -i, or pipe one from ai-img")
	}
	payload := map[string]any{
		"image_url": dataURI(req.InputImage, req.InputMIME),
		"pbr":       req.PBR,
	}
	if req.FaceCount > 0 {
		payload["face_limit"] = req.FaceCount
	}
	return payload, nil
}

func (t *Tripo) Submit(ctx context.Context, req *GenerateRequest) (string, error) {
	payload, err := t.payload(req)
	if err != nil {
		return "", err
	}
	s, err := t.submit(ctx, req.APIKey, modelOr(req.Model, t.DefaultModel()), payload)
	if err != nil {
		return "", err
	}
	return s.RequestID, nil
}

func (t *Tripo) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	payload, err := t.payload(req)
	if err != nil {
		return nil, err
	}
	return t.run(ctx, req.APIKey, modelOr(req.Model, t.DefaultModel()), payload)
}

func (t *Tripo) Fetch(ctx context.Context, apiKey, requestID string) (*GenerateResponse, error) {
	return t.fetch(ctx, apiKey, "tripo3d/tripo", requestID)
}
