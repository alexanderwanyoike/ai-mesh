package provider

// falPixal3DPayload builds the request for TencentARC Pixal3D on Fal. Pixal3D
// controls poly count via decimation_target (remesh on); --faces maps to it, and
// --pbr has no effect (Pixal3D always textures).
func falPixal3DPayload(req *GenerateRequest) (map[string]any, error) {
	payload := map[string]any{
		"image_url": dataURI(req.InputImage, req.InputMIME),
	}
	if req.FaceCount > 0 {
		payload["remesh"] = true
		payload["decimation_target"] = req.FaceCount
	}
	return payload, nil
}
