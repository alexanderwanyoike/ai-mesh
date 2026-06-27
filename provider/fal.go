package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
)

func init() { Register(&Fal{}) }

const falDefaultModel = "fal-ai/hunyuan-3d/v3.1/pro/image-to-3d"

// Fal implements Provider using Fal's queue API (Tencent Hunyuan3D 3.1).
// It is image-to-3d only.
type Fal struct {
	HTTPClient *http.Client // overridable for tests
	BaseURL    string       // overridable for tests; default https://queue.fal.run
}

func (f *Fal) Name() string         { return "fal" }
func (f *Fal) DefaultModel() string { return falDefaultModel }
func (f *Fal) APIKeyEnv() string    { return "FAL_API_KEY" }
func (f *Fal) APIKeyURL() string    { return "https://fal.ai/dashboard/keys" }

func (f *Fal) httpClient() *http.Client {
	if f.HTTPClient != nil {
		return f.HTTPClient
	}
	return http.DefaultClient
}

func (f *Fal) baseURL() string {
	if f.BaseURL != "" {
		return f.BaseURL
	}
	return "https://queue.fal.run"
}

type falSubmitResponse struct {
	StatusURL   string `json:"status_url"`
	ResponseURL string `json:"response_url"`
}

type falStatusResponse struct {
	Status string `json:"status"`
}

type falFile struct {
	URL string `json:"url"`
}

type falResult struct {
	ModelGLB  falFile `json:"model_glb"`
	ModelURLs struct {
		GLB falFile `json:"glb"`
	} `json:"model_urls"`
	Seed int `json:"seed"`
}

func (f *Fal) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	if req.APIKey == "" {
		return nil, fmt.Errorf("no API key provided for fal")
	}
	if req.InputImage == nil {
		return nil, fmt.Errorf("fal is image-to-3d only: pass an image with -i, pipe one from ai-img, or use -p meshy for text-to-3d")
	}

	model := req.Model
	if model == "" {
		model = f.DefaultModel()
	}
	headers := map[string]string{"Authorization": "Key " + req.APIKey}

	payload := map[string]any{
		"input_image_url": dataURI(req.InputImage, req.InputMIME),
		"enable_pbr":      req.PBR,
	}
	if req.FaceCount > 0 {
		payload["face_count"] = req.FaceCount
	}

	var submit falSubmitResponse
	if err := postJSON(ctx, f.httpClient(), f.baseURL()+"/"+model, headers, payload, &submit); err != nil {
		return nil, fmt.Errorf("submitting job: %w", err)
	}
	if submit.StatusURL == "" || submit.ResponseURL == "" {
		return nil, fmt.Errorf("fal submit returned no queue URLs")
	}

	if err := poll(ctx, func() (bool, error) {
		var status falStatusResponse
		if err := getJSON(ctx, f.httpClient(), submit.StatusURL, headers, &status); err != nil {
			return false, fmt.Errorf("polling status: %w", err)
		}
		switch status.Status {
		case "COMPLETED":
			return true, nil
		case "IN_QUEUE", "IN_PROGRESS":
			return false, nil
		default:
			return false, fmt.Errorf("fal job in unexpected state %q", status.Status)
		}
	}); err != nil {
		return nil, err
	}

	var result falResult
	if err := getJSON(ctx, f.httpClient(), submit.ResponseURL, headers, &result); err != nil {
		return nil, fmt.Errorf("fetching result: %w", err)
	}

	// Prefer the dedicated GLB url. model_glb is polymorphic - on some tiers
	// (e.g. rapid) it can point at an OBJ - whereas model_urls.glb is always GLB.
	url := result.ModelURLs.GLB.URL
	if url == "" {
		url = result.ModelGLB.URL
	}
	if url == "" {
		return nil, fmt.Errorf("fal response had no GLB URL")
	}

	data, err := download(ctx, f.httpClient(), url)
	if err != nil {
		return nil, err
	}
	return &GenerateResponse{ModelData: data, Format: "glb", URL: url}, nil
}

// dataURI encodes image bytes as a base64 data URI accepted by Fal and Meshy.
func dataURI(image []byte, mime string) string {
	if mime == "" {
		mime = "image/png"
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(image)
}
