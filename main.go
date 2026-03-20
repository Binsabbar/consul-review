// Package main is the entry point for the consul-review CLI.
package main

import "github.com/binsabbar/consul-review/cmd"

// Build info injected at compile time via -ldflags.
// Populated by GoReleaser and `make go-build-release`.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "manual"
)

func main() {
	cmd.SetVersionInfo(version, commit, date, builtBy)
	cmd.Execute()
}
