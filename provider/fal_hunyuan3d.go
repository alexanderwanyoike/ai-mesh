package provider

// falHunyuanPayload builds the request for Tencent Hunyuan3D 3.1 on Fal. It is
// also the payload used for raw-endpoint passthrough, since input_image_url is
// the most common Fal image-to-3d shape.
func falHunyuanPayload(req *GenerateRequest) (map[string]any, error) {
	payload := map[string]any{
		"input_image_url": dataURI(req.InputImage, req.InputMIME),
		"enable_pbr":      req.PBR,
	}
	if req.FaceCount > 0 {
		payload["face_count"] = req.FaceCount
	}
	return payload, nil
}
