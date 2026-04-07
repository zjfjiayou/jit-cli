package main

import (
	"context"
	"os"

	"jit-cli/cmd"
	"jit-cli/internal/build"
)

var (
	version = ""
	commit  = ""
	date    = ""
)

func main() {
	if version != "" {
		build.Version = version
	}
	if commit != "" {
		build.Commit = commit
	}
	if date != "" {
		build.Date = date
	}
	os.Exit(cmd.Execute(context.Background()))
}
