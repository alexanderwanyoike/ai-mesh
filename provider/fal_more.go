package provider

import (
	"context"
	"fmt"
	"net/http"
)

func init() {
	Register(&Pixal3D{})
	Register(&Tripo{})
	Register(&Rodin{})
}

// falQueue is the shared Fal queue-API transport for the Fal-hosted providers
// beyond Hunyuan3D (Pixal3D, Tripo, Rodin). They all submit to
// https://queue.fal.run/{model} with "Key" auth, poll the returned status URL
// until COMPLETED, then download the GLB; only the endpoint, request payload,
// and the app base used to reconstruct fetch-by-id URLs differ. Every Fal
// provider reads the same FAL_API_KEY. Concrete providers embed this and supply
// their payload.
type falQueue struct {
	HTTPClient *http.Client // overridable for tests
	BaseURL    string       // overridable for tests; default https://queue.fal.run
}

func (q *falQueue) httpClient() *http.Client {
	if q.HTTPClient != nil {
		return q.HTTPClient
	}
	return http.DefaultClient
}

func (q *falQueue) baseURL() string {
	if q.BaseURL != "" {
		return q.BaseURL
	}
	return "https://queue.fal.run"
}

// submit posts payload to the model endpoint and returns Fal's queue handles.
func (q *falQueue) submit(ctx context.Context, apiKey, model string, payload map[string]any) (falSubmitResponse, error) {
	var submit falSubmitResponse
	if err := postJSON(ctx, q.httpClient(), q.baseURL()+"/"+model, falKeyHeader(apiKey), payload, &submit); err != nil {
		return submit, fmt.Errorf("submitting job: %w", err)
	}
	if submit.RequestID == "" {
		return submit, fmt.Errorf("fal submit returned no request id")
	}
	return submit, nil
}

// run submits the job and waits for the finished model.
func (q *falQueue) run(ctx context.Context, apiKey, model string, payload map[string]any) (*GenerateResponse, error) {
	s, err := q.submit(ctx, apiKey, model, payload)
	if err != nil {
		return nil, err
	}
	if s.StatusURL == "" || s.ResponseURL == "" {
		return nil, fmt.Errorf("fal submit returned no queue URLs")
	}
	return q.await(ctx, apiKey, s.StatusURL, s.ResponseURL)
}

// await polls statusURL until COMPLETED, then downloads the GLB from responseURL.
func (q *falQueue) await(ctx context.Context, apiKey, statusURL, responseURL string) (*GenerateResponse, error) {
	headers := falKeyHeader(apiKey)

	if err := poll(ctx, func() (bool, error) {
		var status falStatusResponse
		if err := getJSON(ctx, q.httpClient(), statusURL, headers, &status); err != nil {
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
	if err := getJSON(ctx, q.httpClient(), responseURL, headers, &result); err != nil {
		return nil, fmt.Errorf("fetching result: %w", err)
	}
	url := falGLBURL(result)
	if url == "" {
		return nil, fmt.Errorf("fal response had no GLB URL")
	}
	data, err := download(ctx, q.httpClient(), url)
	if err != nil {
		return nil, err
	}
	return &GenerateResponse{ModelData: data, Format: "glb", URL: url}, nil
}

// fetch waits for an already-submitted job (by request ID). Fal's per-request
// endpoints live under the queue "app" base (.../{appBase}/requests/{id}), not
// the full versioned model path, so callers pass the app base.
func (q *falQueue) fetch(ctx context.Context, apiKey, appBase, requestID string) (*GenerateResponse, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("no API key provided")
	}
	base := q.baseURL() + "/" + appBase + "/requests/" + requestID
	return q.await(ctx, apiKey, base+"/status", base)
}

// modelOr returns the request's model override, or the provider default.
func modelOr(model, def string) string {
	if model != "" {
		return model
	}
	return def
}

// --- Pixal3D ----------------------------------------------------------------

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

// --- Tripo ------------------------------------------------------------------

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

// --- Rodin ------------------------------------------------------------------

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
