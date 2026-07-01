package provider

import (
	"context"
	"fmt"
	"net/http"
)

// falQueue is the shared Fal queue-API transport used by every Fal-hosted
// provider (Hunyuan3D, Pixal3D, Tripo, Rodin). They all submit to
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
	ModelMesh falFile `json:"model_mesh"`
	ModelURLs struct {
		GLB falFile `json:"glb"`
	} `json:"model_urls"`
	Seed int `json:"seed"`
}

// falGLBURL resolves the GLB download url across the output shapes Fal's 3D
// models use: model_urls.glb (Hunyuan3D), model_glb (Hunyuan3D, Pixal3D), or
// model_mesh (Tripo, Rodin). model_urls.glb wins because model_glb is
// polymorphic - on some Hunyuan tiers (e.g. rapid) it points at an OBJ, whereas
// model_urls.glb is always GLB.
func falGLBURL(r falResult) string {
	if r.ModelURLs.GLB.URL != "" {
		return r.ModelURLs.GLB.URL
	}
	if r.ModelGLB.URL != "" {
		return r.ModelGLB.URL
	}
	return r.ModelMesh.URL
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

func falKeyHeader(apiKey string) map[string]string {
	return map[string]string{"Authorization": "Key " + apiKey}
}

// modelOr returns the request's model override, or the provider default.
func modelOr(model, def string) string {
	if model != "" {
		return model
	}
	return def
}
