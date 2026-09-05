package internal

import (
	"context"
	"fmt"
	"io"
)

// Main starts the combined executable through the initialized application router.
func Main(ctx context.Context, version string, arguments []string, stdout, stderr io.Writer) error {
	router := InitRouter(version, stdout, stderr)
	return router.Run(ctx, arguments)
}

// CLIMain starts the explicit command-line interface.
func CLIMain(ctx context.Context, version string, arguments []string, stdout, stderr io.Writer) error {
	command := InitCLIRouter(version, stdout, stderr)
	return executeCLI(ctx, command, arguments)
}

// ProviderMain starts only the Terraform Plugin Protocol server.
func ProviderMain(ctx context.Context, version string, arguments []string) error {
	router := InitProviderRouter(version)
	return router.Run(ctx, arguments)
}

// ReportError writes a command-boundary error without exposing source content.
func ReportError(stderr io.Writer, err error) {
	if stderr == nil || err == nil {
		return
	}
	_, _ = fmt.Fprintln(stderr, err)
}
