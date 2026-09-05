package controller

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

type ServeFunc func(command *cobra.Command, debug bool) error

type CommonController interface {
	VersionCommand() *cobra.Command
	ServeCommand() *cobra.Command
}

type commonController struct {
	version string
	serve   ServeFunc
}

func NewCommonController(version string, serve ServeFunc) CommonController {
	return &commonController{version: version, serve: serve}
}

func IsCLIInvocation(arguments []string) bool {
	if len(arguments) == 0 {
		return false
	}
	switch arguments[0] {
	case "convert", "version", "serve", "help", "completion", "-h", "--help", "--version":
		return true
	default:
		return false
	}
}

func (rcvr *commonController) VersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print provider version",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			_, err := fmt.Fprintln(command.OutOrStdout(), rcvr.version)
			return err
		},
	}
}

func (rcvr *commonController) ServeCommand() *cobra.Command {
	var debug bool
	command := &cobra.Command{
		Use:   "serve",
		Short: "Explicitly start the Terraform Plugin Protocol v6 server",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			if rcvr.serve == nil {
				return errors.New("provider server is not configured")
			}
			return rcvr.serve(command, debug)
		},
	}
	command.Flags().BoolVar(&debug, "debug", false, "run with managed debugger support")
	return command
}
