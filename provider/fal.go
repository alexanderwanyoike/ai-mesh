package provider

import (
	"context"
	"fmt"
)

func init() { Register(&Fal{}) }

const falDefaultModel = "fal-ai/hunyuan-3d/v3.1/pro/image-to-3d"

// falAppBase is the queue "app" the Hunyuan3D models share. Fal's per-request
// endpoints live here (e.g. .../fal-ai/hunyuan-3d/requests/{id}), not under the
// full versioned model path, so fetch-by-id reconstructs URLs from it.
const falAppBase = "fal-ai/hunyuan-3d"

// Fal implements Provider using Fal's queue API (Tencent Hunyuan3D 3.1).
// It is image-to-3d only.
type Fal struct{ falQueue }

func (f *Fal) Name() string         { return "fal" }
func (f *Fal) DefaultModel() string { return falDefaultModel }
func (f *Fal) APIKeyEnv() string    { return "FAL_API_KEY" }
func (f *Fal) APIKeyURL() string    { return "https://fal.ai/dashboard/keys" }

func (f *Fal) payload(req *GenerateRequest) (map[string]any, error) {
	if req.APIKey == "" {
		return nil, fmt.Errorf("no API key provided for fal")
	}
	if req.InputImage == nil {
		return nil, fmt.Errorf("fal is image-to-3d only: pass an image with -i, pipe one from ai-img, or use -p meshy for text-to-3d")
	}
	payload := map[string]any{
		"input_image_url": dataURI(req.InputImage, req.InputMIME),
		"enable_pbr":      req.PBR,
	}
	if req.FaceCount > 0 {
		payload["face_count"] = req.FaceCount
	}
	return payload, nil
}

func (f *Fal) Submit(ctx context.Context, req *GenerateRequest) (string, error) {
	payload, err := f.payload(req)
	if err != nil {
		return "", err
	}
	s, err := f.submit(ctx, req.APIKey, modelOr(req.Model, f.DefaultModel()), payload)
	if err != nil {
		return "", err
	}
	return s.RequestID, nil
}

func (f *Fal) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	payload, err := f.payload(req)
	if err != nil {
		return nil, err
	}
	return f.run(ctx, req.APIKey, modelOr(req.Model, f.DefaultModel()), payload)
}

func (f *Fal) Fetch(ctx context.Context, apiKey, requestID string) (*GenerateResponse, error) {
	return f.fetch(ctx, apiKey, falAppBase, requestID)
}
