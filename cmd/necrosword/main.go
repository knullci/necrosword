package main

import (
	"fmt"
	"os"

	"github.com/knullci/necrosword/internal/app"
	"github.com/knullci/necrosword/internal/config"
	"github.com/spf13/cobra"
)

var (
	version   = "0.1.0"
	commit    = "development"
	buildDate = "unknown"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "necrosword",
		Short: "Necrosword - High-performance process executor for Knull CI/CD",
		Long: `Necrosword is a blazing-fast process executor written in Go.
It handles build pipeline execution, command running, and real-time output streaming
for the Knull CI/CD platform.`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// Version command
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Necrosword v%s\n", version)
			fmt.Printf("Commit: %s\n", commit)
			fmt.Printf("Built: %s\n", buildDate)
		},
	}

	// Server command - starts the gRPC server
	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "Start the Necrosword gRPC server",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			application, err := app.New(cfg)
			if err != nil {
				return fmt.Errorf("failed to create application: %w", err)
			}

			return application.Run()
		},
	}

	// Execute command - runs a single command
	executeCmd := &cobra.Command{
		Use:   "execute",
		Short: "Execute a single command",
		Example: `  necrosword execute --tool git --args "clone,https://github.com/user/repo.git"
  necrosword execute --tool npm --args "install" --workdir /path/to/project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			tool, _ := cmd.Flags().GetString("tool")
			cmdArgs, _ := cmd.Flags().GetString("args")
			workdir, _ := cmd.Flags().GetString("workdir")

			if tool == "" {
				return fmt.Errorf("tool is required")
			}

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			application, err := app.New(cfg)
			if err != nil {
				return fmt.Errorf("failed to create application: %w", err)
			}

			return application.ExecuteCommand(tool, cmdArgs, workdir)
		},
	}

	executeCmd.Flags().StringP("tool", "t", "", "Tool to execute (git, npm, mvn, docker, kubectl)")
	executeCmd.Flags().StringP("args", "a", "", "Comma-separated arguments")
	executeCmd.Flags().StringP("workdir", "w", ".", "Working directory")

	rootCmd.AddCommand(versionCmd, serverCmd, executeCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
