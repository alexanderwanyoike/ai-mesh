package provider

import (
	"context"
	"fmt"
	"net/http"
)

func init() { Register(&Meshy{}) }

const meshyDefaultModel = "meshy-5"

// Meshy implements Provider using the Meshy OpenAPI. It supports both
// image-to-3d (single stage) and text-to-3d (preview then refine).
type Meshy struct {
	HTTPClient *http.Client // overridable for tests
	BaseURL    string       // overridable for tests; default https://api.meshy.ai
}

func (m *Meshy) Name() string         { return "meshy" }
func (m *Meshy) DefaultModel() string { return meshyDefaultModel }
func (m *Meshy) APIKeyEnv() string    { return "MESHY_API_KEY" }
func (m *Meshy) APIKeyURL() string    { return "https://www.meshy.ai/api" }

func (m *Meshy) httpClient() *http.Client {
	if m.HTTPClient != nil {
		return m.HTTPClient
	}
	return http.DefaultClient
}

func (m *Meshy) baseURL() string {
	if m.BaseURL != "" {
		return m.BaseURL
	}
	return "https://api.meshy.ai"
}

type meshyCreateResponse struct {
	Result string `json:"result"`
}

type meshyTask struct {
	Status    string `json:"status"`
	ModelURLs struct {
		GLB string `json:"glb"`
	} `json:"model_urls"`
	TaskError struct {
		Message string `json:"message"`
	} `json:"task_error"`
	ConsumedCredits int `json:"consumed_credits"`
}

func (m *Meshy) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	if req.APIKey == "" {
		return nil, fmt.Errorf("no API key provided for meshy")
	}

	aiModel := req.Model
	if aiModel == "" {
		aiModel = m.DefaultModel()
	}

	var task *meshyTask
	var err error
	if req.InputImage != nil {
		task, err = m.imageTo3D(ctx, req.APIKey, aiModel, req)
	} else {
		task, err = m.textTo3D(ctx, req.APIKey, aiModel, req)
	}
	if err != nil {
		return nil, err
	}

	if task.ModelURLs.GLB == "" {
		return nil, fmt.Errorf("meshy task had no GLB URL")
	}
	data, err := download(ctx, m.httpClient(), task.ModelURLs.GLB)
	if err != nil {
		return nil, err
	}
	return &GenerateResponse{ModelData: data, Format: "glb", URL: task.ModelURLs.GLB}, nil
}

func (m *Meshy) imageTo3D(ctx context.Context, apiKey, aiModel string, req *GenerateRequest) (*meshyTask, error) {
	headers := m.headers(apiKey)
	payload := map[string]any{
		"image_url":      dataURI(req.InputImage, req.InputMIME),
		"ai_model":       aiModel,
		"should_texture": true,
		"enable_pbr":     req.PBR,
		"target_formats": []string{"glb"},
	}
	if req.FaceCount > 0 {
		payload["target_polycount"] = req.FaceCount
	}

	var created meshyCreateResponse
	if err := postJSON(ctx, m.httpClient(), m.baseURL()+"/openapi/v1/image-to-3d", headers, payload, &created); err != nil {
		return nil, fmt.Errorf("creating image-to-3d task: %w", err)
	}
	if created.Result == "" {
		return nil, fmt.Errorf("meshy create returned no task id")
	}
	return m.waitForTask(ctx, headers, m.baseURL()+"/openapi/v1/image-to-3d/"+created.Result)
}

func (m *Meshy) textTo3D(ctx context.Context, apiKey, aiModel string, req *GenerateRequest) (*meshyTask, error) {
	headers := m.headers(apiKey)
	textURL := m.baseURL() + "/openapi/v2/text-to-3d"

	// Stage 1: preview (geometry).
	preview := map[string]any{
		"mode":           "preview",
		"prompt":         req.Prompt,
		"ai_model":       aiModel,
		"target_formats": []string{"glb"},
	}
	if req.FaceCount > 0 {
		preview["target_polycount"] = req.FaceCount
	}
	var previewCreated meshyCreateResponse
	if err := postJSON(ctx, m.httpClient(), textURL, headers, preview, &previewCreated); err != nil {
		return nil, fmt.Errorf("creating text-to-3d preview: %w", err)
	}
	if previewCreated.Result == "" {
		return nil, fmt.Errorf("meshy preview returned no task id")
	}
	if _, err := m.waitForTask(ctx, headers, textURL+"/"+previewCreated.Result); err != nil {
		return nil, err
	}

	// Stage 2: refine (texture) using the preview task id.
	refine := map[string]any{
		"mode":            "refine",
		"preview_task_id": previewCreated.Result,
		"ai_model":        aiModel,
		"enable_pbr":      req.PBR,
		"target_formats":  []string{"glb"},
	}
	var refineCreated meshyCreateResponse
	if err := postJSON(ctx, m.httpClient(), textURL, headers, refine, &refineCreated); err != nil {
		return nil, fmt.Errorf("creating text-to-3d refine: %w", err)
	}
	if refineCreated.Result == "" {
		return nil, fmt.Errorf("meshy refine returned no task id")
	}
	return m.waitForTask(ctx, headers, textURL+"/"+refineCreated.Result)
}

// waitForTask polls a Meshy task URL until it SUCCEEDED and returns it.
func (m *Meshy) waitForTask(ctx context.Context, headers map[string]string, taskURL string) (*meshyTask, error) {
	var task meshyTask
	err := poll(ctx, func() (bool, error) {
		task = meshyTask{}
		if err := getJSON(ctx, m.httpClient(), taskURL, headers, &task); err != nil {
			return false, fmt.Errorf("polling task: %w", err)
		}
		switch task.Status {
		case "SUCCEEDED":
			return true, nil
		case "PENDING", "IN_PROGRESS":
			return false, nil
		case "FAILED", "CANCELED":
			return false, fmt.Errorf("meshy task %s: %s", task.Status, task.TaskError.Message)
		default:
			return false, nil
		}
	})
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (m *Meshy) headers(apiKey string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + apiKey}
}
