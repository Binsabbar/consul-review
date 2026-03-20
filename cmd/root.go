// Package cmd provides the consul-review CLI commands.
package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// versionInfo holds build-time metadata injected via -ldflags.
var versionInfo struct {
	version string
	commit  string
	date    string
	builtBy string
}

// SetVersionInfo is called from main() to inject ldflags build metadata.
func SetVersionInfo(version, commit, date, builtBy string) {
	versionInfo.version = version
	versionInfo.commit = commit
	versionInfo.date = date
	versionInfo.builtBy = builtBy
}

// rootCmd is the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "consul-review",
	Short: "Orchestrate multiple AI agents for automated PR code reviews",
	Long: `consul-review fans out PR review requests to multiple AI agents (Consuls)
in parallel, then aggregates their outputs into a single consolidated review.`,
	Version: "dev", // overridden by SetVersionInfo → cobra --version flag
}

// Execute is the entry point called from main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(
		&cfgFile,
		"config",
		"",
		"config file (default: ~/.consul-review/config.yaml)",
	)

	rootCmd.PersistentFlags().Bool("debug", false, "enable debug logging")
	_ = viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
}

// initConfig is called after flags are parsed.
func initConfig() {
	if viper.GetBool("debug") {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	// Wire version string into rootCmd now that SetVersionInfo has been called.
	if versionInfo.version != "" && versionInfo.version != "dev" {
		rootCmd.Version = fmt.Sprintf(
			"%s (commit=%s date=%s builtBy=%s)",
			versionInfo.version, versionInfo.commit, versionInfo.date, versionInfo.builtBy,
		)
	}
}
