package provider

// falRodinPayload builds the request for Hyper3D Rodin on Fal. It takes an array
// of image urls (multi-view); a single reference is the common case. --pbr
// selects PBR vs Shaded material; Rodin sets density via its own quality tier, so
// --faces has no effect.
func falRodinPayload(req *GenerateRequest) (map[string]any, error) {
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
