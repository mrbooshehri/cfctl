package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mrbooshehri/cfctl/config"
	"github.com/mrbooshehri/cfctl/tui"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cfctl",
	Short: "Cloudflare TUI manager",
	Long:  "An interactive terminal UI to manage your Cloudflare zones, DNS, firewall, and SSL.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		model, err := tui.New(cfg)
		if err != nil {
			return fmt.Errorf("initializing app: %w", err)
		}

		p := tea.NewProgram(model, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return err
		}
		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("cfctl v0.1.0")
	},
}

func Execute() {
	rootCmd.AddCommand(versionCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
