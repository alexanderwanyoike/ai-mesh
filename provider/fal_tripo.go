package provider

// falTripoPayload builds the request for Tripo v2.5 on Fal. --faces maps to
// face_limit; --pbr requests PBR maps.
func falTripoPayload(req *GenerateRequest) (map[string]any, error) {
	payload := map[string]any{
		"image_url": dataURI(req.InputImage, req.InputMIME),
		"pbr":       req.PBR,
	}
	if req.FaceCount > 0 {
		payload["face_limit"] = req.FaceCount
	}
	return payload, nil
}
