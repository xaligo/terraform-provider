package main

import (
	"context"
	"os"

	application "github.com/xaligo/terraform-provider/internal"
)

var version = "dev"

func main() {
	if err := application.ProviderMain(context.Background(), version, os.Args[1:]); err != nil {
		application.ReportError(os.Stderr, err)
		os.Exit(1)
	}
}
