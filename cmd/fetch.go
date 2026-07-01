package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alexanderwanyoike/ai-mesh/provider"
	"github.com/spf13/cobra"
)

var fetchCmd = &cobra.Command{
	Use:   "fetch <provider> <job-id>",
	Short: "Retrieve a previously submitted job by ID",
	Long: `Fetch waits for an already-submitted job to finish and downloads the GLB.

Use it with a job ID printed by 'ai-mesh --no-wait'. Pass the same -m model you
submitted with, so the job can be routed (Fal uses a per-model app base).`,
	Example: `  ai-mesh fetch fal 019f095f-8728-76d2-9005-2cd19f32a143 -m tripo -o hero.glb`,
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runFetch(args[0], args[1])
	},
}

func init() {
	fetchCmd.Flags().StringVarP(&model, "model", "m", "", "model the job was submitted with (defaults per provider)")
	fetchCmd.Flags().StringVarP(&output, "output", "o", "output.glb", "output file path")
	fetchCmd.Flags().StringVarP(&apiKey, "api-key", "k", "", "API key (overrides env var and config)")
	fetchCmd.Flags().IntVar(&timeoutMin, "timeout", 30, "minutes to wait for the job before giving up")
	rootCmd.AddCommand(fetchCmd)
}

func runFetch(name, jobID string) error {
	p := provider.Get(name)
	if p == nil {
		return fmt.Errorf("unknown provider %q (available: %s)", name, strings.Join(provider.List(), ", "))
	}
	key, err := resolveKey(p)
	if err != nil {
		return err
	}
	provider.SetWaitTimeout(time.Duration(timeoutMin) * time.Minute)

	fmt.Fprintf(os.Stderr, "Fetching %s job %s ...\n", p.Name(), jobID)
	resp, err := p.Fetch(context.Background(), key, model, jobID)
	if err != nil {
		return err
	}
	return writeOutput(resp.ModelData, output)
}
