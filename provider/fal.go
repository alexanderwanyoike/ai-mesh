package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
)

func init() { Register(&Fal{}) }

const falDefaultModel = "fal-ai/hunyuan-3d/v3.1/pro/image-to-3d"

// falAppBase is the queue "app" all ai-mesh Fal models share. Fal's per-request
// endpoints live here (e.g. .../fal-ai/hunyuan-3d/requests/{id}), not under the
// full versioned model path, so fetch-by-id reconstructs URLs from it.
const falAppBase = "fal-ai/hunyuan-3d"

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
	RequestID   string `json:"request_id"`
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

// submit posts the job and returns Fal's queue handles.
func (f *Fal) submit(ctx context.Context, req *GenerateRequest) (falSubmitResponse, error) {
	var submit falSubmitResponse
	if req.APIKey == "" {
		return submit, fmt.Errorf("no API key provided for fal")
	}
	if req.InputImage == nil {
		return submit, fmt.Errorf("fal is image-to-3d only: pass an image with -i, pipe one from ai-img, or use -p meshy for text-to-3d")
	}

	model := req.Model
	if model == "" {
		model = f.DefaultModel()
	}
	payload := map[string]any{
		"input_image_url": dataURI(req.InputImage, req.InputMIME),
		"enable_pbr":      req.PBR,
	}
	if req.FaceCount > 0 {
		payload["face_count"] = req.FaceCount
	}

	if err := postJSON(ctx, f.httpClient(), f.baseURL()+"/"+model, falKeyHeader(req.APIKey), payload, &submit); err != nil {
		return submit, fmt.Errorf("submitting job: %w", err)
	}
	if submit.RequestID == "" {
		return submit, fmt.Errorf("fal submit returned no request id")
	}
	return submit, nil
}

// Submit fires a job and returns its request ID (for --no-wait / fetch).
func (f *Fal) Submit(ctx context.Context, req *GenerateRequest) (string, error) {
	s, err := f.submit(ctx, req)
	if err != nil {
		return "", err
	}
	return s.RequestID, nil
}

// Generate submits and waits for the result.
func (f *Fal) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	s, err := f.submit(ctx, req)
	if err != nil {
		return nil, err
	}
	if s.StatusURL == "" || s.ResponseURL == "" {
		return nil, fmt.Errorf("fal submit returned no queue URLs")
	}
	return f.await(ctx, req.APIKey, s.StatusURL, s.ResponseURL)
}

// Fetch waits for an already-submitted job (by request ID) and returns the result.
func (f *Fal) Fetch(ctx context.Context, apiKey, requestID string) (*GenerateResponse, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("no API key provided for fal")
	}
	base := f.baseURL() + "/" + falAppBase + "/requests/" + requestID
	return f.await(ctx, apiKey, base+"/status", base)
}

// await polls statusURL until COMPLETED, then downloads the GLB from responseURL.
func (f *Fal) await(ctx context.Context, apiKey, statusURL, responseURL string) (*GenerateResponse, error) {
	headers := falKeyHeader(apiKey)

	if err := poll(ctx, func() (bool, error) {
		var status falStatusResponse
		if err := getJSON(ctx, f.httpClient(), statusURL, headers, &status); err != nil {
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
	if err := getJSON(ctx, f.httpClient(), responseURL, headers, &result); err != nil {
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

func falKeyHeader(apiKey string) map[string]string {
	return map[string]string{"Authorization": "Key " + apiKey}
}

// dataURI encodes image bytes as a base64 data URI accepted by Fal and Meshy.
func dataURI(image []byte, mime string) string {
	if mime == "" {
		mime = "image/png"
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(image)
}
