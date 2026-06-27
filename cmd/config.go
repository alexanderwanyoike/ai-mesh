package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/alexanderwanyoike/ai-mesh/config"
	"github.com/alexanderwanyoike/ai-mesh/provider"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage stored API keys",
	Long: `Store API keys in a config file so they do not have to live in the shell
environment. Keys resolve with precedence: --api-key flag > env var > config file.`,
}

var configSetCmd = &cobra.Command{
	Use:   "set <provider> <api-key>",
	Short: "Store the API key for a provider",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if provider.Get(name) == nil {
			return fmt.Errorf("unknown provider %q (available: %s)", name, strings.Join(provider.List(), ", "))
		}
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		cfg.Keys[name] = args[1]
		if err := config.Save(cfg); err != nil {
			return err
		}
		path, _ := config.Path()
		fmt.Fprintf(os.Stderr, "Saved %s API key to %s\n", name, path)
		return nil
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <provider>",
	Short: "Show the stored API key for a provider (masked)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		fmt.Println(maskKey(cfg.Keys[args[0]]))
		return nil
	},
}

var configUnsetCmd = &cobra.Command{
	Use:   "unset <provider>",
	Short: "Remove the stored API key for a provider",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		delete(cfg.Keys, args[0])
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Removed %s API key\n", args[0])
		return nil
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List API key status for each provider",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		for _, name := range provider.List() {
			p := provider.Get(name)
			stored := cfg.Keys[name]
			line := fmt.Sprintf("%-8s %s", name, maskKey(stored))
			if stored == "" && os.Getenv(p.APIKeyEnv()) != "" {
				line += fmt.Sprintf(" (set via %s)", p.APIKeyEnv())
			}
			fmt.Println(line)
		}
		return nil
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print the config file path",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := config.Path()
		if err != nil {
			return err
		}
		fmt.Println(path)
		return nil
	},
}

func init() {
	configCmd.AddCommand(configSetCmd, configGetCmd, configUnsetCmd, configListCmd, configPathCmd)
	rootCmd.AddCommand(configCmd)
}

// maskKey returns a privacy-preserving representation of an API key.
func maskKey(key string) string {
	if key == "" {
		return "(not set)"
	}
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}
