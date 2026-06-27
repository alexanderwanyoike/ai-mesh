package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	noWait       bool
	timeoutMin   int
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
	rootCmd.Flags().BoolVar(&noWait, "no-wait", false, "submit the job, print its ID, and exit (fetch later with 'ai-mesh fetch')")
	rootCmd.Flags().IntVar(&timeoutMin, "timeout", 30, "minutes to wait for a job before giving up")
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

	key, err := resolveKey(p)
	if err != nil {
		return err
	}
	provider.SetWaitTimeout(time.Duration(timeoutMin) * time.Minute)

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

	ctx := context.Background()

	// Fire-and-forget: submit, print the job ID, and exit without waiting.
	if noWait {
		id, err := p.Submit(ctx, req)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Submitted %s job %s. Fetch it when ready:\n  ai-mesh fetch %s %s -o %s\n",
			p.Name(), id, p.Name(), id, output)
		fmt.Println(id)
		return nil
	}

	// Progress goes to stderr so stdout stays clean for scripting/agents.
	if input != "" {
		fmt.Fprintf(os.Stderr, "Generating mesh from image: %s (%s)\n", input, req.InputMIME)
	} else {
		fmt.Fprintf(os.Stderr, "Generating mesh from prompt: %q\n", prompt)
	}
	fmt.Fprintf(os.Stderr, "Provider: %s | Model: %s\n", p.Name(), modelName)

	resp, err := p.Generate(ctx, req)
	if err != nil {
		return err
	}
	if resp.Message != "" {
		fmt.Fprintf(os.Stderr, "Note: %s\n", resp.Message)
	}
	return writeOutput(resp.ModelData, output)
}

// resolveKey returns the API key for a provider using precedence flag > env > config.
func resolveKey(p provider.Provider) (string, error) {
	cfg, err := config.Load()
	if err != nil {
		return "", err
	}
	key := config.ResolveKey(apiKey, os.Getenv(p.APIKeyEnv()), cfg.Keys[p.Name()])
	if key == "" {
		return "", fmt.Errorf("no API key for provider %q. Set one with any of:\n"+
			"  ai-mesh config set %s <key>\n"+
			"  export %s=<key>\n"+
			"  --api-key <key>\n"+
			"Get a key: %s",
			p.Name(), p.Name(), p.APIKeyEnv(), p.APIKeyURL())
	}
	return key, nil
}

// writeOutput writes the model bytes to path (ensuring a .glb suffix) and prints
// the final path to stdout for scripting.
func writeOutput(data []byte, path string) error {
	if !strings.HasSuffix(strings.ToLower(path), ".glb") {
		path += ".glb"
	}
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating directory: %w", err)
		}
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Saved: %s (%d bytes)\n", path, len(data))
	fmt.Println(path)
	return nil
}
