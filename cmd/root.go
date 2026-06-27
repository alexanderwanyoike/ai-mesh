package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexanderwanyoike/ai-mesh/config"
	"github.com/alexanderwanyoike/ai-mesh/provider"
	"github.com/spf13/cobra"

	// Register providers
	_ "github.com/alexanderwanyoike/ai-mesh/provider"
)

var (
	providerName string
	model        string
	output       string
	input        string
	faces        int
	pbr          bool
	apiKey       string
)

var rootCmd = &cobra.Command{
	Use:   "ai-mesh [flags] [\"prompt\"]",
	Short: "Generate 3D meshes from text or images using AI",
	Long: `ai-mesh generates 3D meshes (GLB) from a text prompt or a reference image.
It downloads the result from the provider and writes it to a file.

Providers:
  fal     - Tencent Hunyuan3D 3.1 on Fal, image-to-3d only (requires FAL_API_KEY)
  meshy   - Meshy, text-to-3d and image-to-3d (requires MESHY_API_KEY)

Compose it with ai-img for text -> image -> mesh:
  ai-img "a stone golem" -o golem.png && ai-mesh -i golem.png -o golem.glb`,
	Example: `  ai-mesh -i character.png -o character.glb
  ai-mesh -p meshy "a low-poly treasure chest" -o chest.glb
  ai-mesh -i sketch.png --faces 100000 --pbr -o sketch.glb`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		prompt := ""
		if len(args) == 1 {
			prompt = args[0]
		}
		return run(prompt)
	},
}

func init() {
	rootCmd.Flags().StringVarP(&providerName, "provider", "p", "fal", "provider to use: fal, meshy")
	rootCmd.Flags().StringVarP(&model, "model", "m", "", "model name/ID (defaults per provider)")
	rootCmd.Flags().StringVarP(&output, "output", "o", "output.glb", "output file path")
	rootCmd.Flags().StringVarP(&input, "input", "i", "", "input image for image-to-3d")
	rootCmd.Flags().IntVar(&faces, "faces", 50000, "target face/polygon count")
	rootCmd.Flags().BoolVar(&pbr, "pbr", false, "request PBR material maps")
	rootCmd.Flags().StringVarP(&apiKey, "api-key", "k", "", "API key (overrides env var and config)")
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(prompt string) error {
	p := provider.Get(providerName)
	if p == nil {
		return fmt.Errorf("unknown provider %q (available: %s)", providerName, strings.Join(provider.List(), ", "))
	}

	if prompt == "" && input == "" {
		return fmt.Errorf("provide a text prompt or an --input image")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	key := config.ResolveKey(apiKey, os.Getenv(p.APIKeyEnv()), cfg.Keys[p.Name()])
	if key == "" {
		return fmt.Errorf("no API key for provider %q. Set one with any of:\n"+
			"  ai-mesh config set %s <key>\n"+
			"  export %s=<key>\n"+
			"  --api-key <key>\n"+
			"Get a key: %s",
			p.Name(), p.Name(), p.APIKeyEnv(), p.APIKeyURL())
	}

	req := &provider.GenerateRequest{
		APIKey:    key,
		Prompt:    prompt,
		Model:     model,
		FaceCount: faces,
		PBR:       pbr,
	}

	if input != "" {
		imageData, err := os.ReadFile(input)
		if err != nil {
			return fmt.Errorf("reading input image: %w", err)
		}
		req.InputImage = imageData
		req.InputMIME = provider.DetectMIMEType(input)
	}

	modelName := model
	if modelName == "" {
		modelName = p.DefaultModel()
	}

	// Progress goes to stderr so stdout stays clean for scripting/agents.
	if input != "" {
		fmt.Fprintf(os.Stderr, "Generating mesh from image: %s (%s)\n", input, req.InputMIME)
	} else {
		fmt.Fprintf(os.Stderr, "Generating mesh from prompt: %q\n", prompt)
	}
	fmt.Fprintf(os.Stderr, "Provider: %s | Model: %s\n", p.Name(), modelName)

	ctx := context.Background()
	resp, err := p.Generate(ctx, req)
	if err != nil {
		return err
	}
	if resp.Message != "" {
		fmt.Fprintf(os.Stderr, "Note: %s\n", resp.Message)
	}

	outputFile := output
	if !strings.HasSuffix(strings.ToLower(outputFile), ".glb") {
		outputFile += ".glb"
	}

	dir := filepath.Dir(outputFile)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating directory: %w", err)
		}
	}

	if err := os.WriteFile(outputFile, resp.ModelData, 0644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Saved: %s (%d bytes)\n", outputFile, len(resp.ModelData))
	// The final path on stdout, so `ai-mesh ... | xargs blender ...` works.
	fmt.Println(outputFile)
	return nil
}
