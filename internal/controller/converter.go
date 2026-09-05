package controller

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	commonentity "github.com/xaligo/terraform-provider/internal/entity/common"
	xaligoentity "github.com/xaligo/terraform-provider/internal/entity/xaligo"
	"github.com/xaligo/terraform-provider/internal/usecase"
)

type ConverterController interface {
	Command() *cobra.Command
}

type converterController struct {
	diagrams usecase.DiagramUsecase
}

func NewConverterController(diagrams usecase.DiagramUsecase) ConverterController {
	return &converterController{diagrams: diagrams}
}

func (rcvr *converterController) Command() *cobra.Command {
	options := xaligoentity.ConvertOptions{}
	command := &cobra.Command{
		Use:   "convert <source-dir>",
		Short: "Convert direct .tf files into canonical XAL V1",
		Long: `Read regular .tf files directly inside source-dir, build a conservative
static infrastructure graph, and atomically write canonical XAL V1. The command
does not invoke Terraform, initialize providers, read state, or contact cloud APIs.
A relative output path is resolved from source-dir.`,
		Example: "  terraform-provider-xaligo convert ./terraform -o architecture.xal",
		Args:    cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, arguments []string) error {
			if rcvr.diagrams == nil {
				return errors.New("diagram use case is not configured")
			}
			result, err := rcvr.diagrams.Generate(command.Context(), commonentity.DiagramSpec{
				SourceDir:     arguments[0],
				OutputPath:    options.Output,
				FrameID:       options.FrameID,
				Title:         options.Title,
				PaperSize:     options.PaperSize,
				Orientation:   options.Orientation,
				GridColumns:   options.GridColumns,
				GridGap:       options.GridGap,
				FailOnWarning: options.FailOnWarning,
				Overwrite:     options.Overwrite,
			})
			printDiagnostics(command.ErrOrStderr(), result.Diagnostics)
			if err != nil {
				if errors.Is(err, usecase.ErrConversion) {
					return errors.New("conversion failed")
				}
				return err
			}
			_, err = fmt.Fprintf(command.OutOrStdout(), "generated %s\n", result.OutputPath)
			return err
		},
	}
	command.Flags().StringVarP(&options.Output, "output", "o", "", "output .xal file path (required)")
	command.Flags().StringVar(&options.FrameID, "frame-id", "main", "XAL frame id")
	command.Flags().StringVar(&options.Title, "title", "", "optional frame title")
	command.Flags().StringVar(&options.PaperSize, "paper-size", "screen", "frame paper size: screen, A5, A4, A3, A2, A1, Letter, Legal, or Tabloid")
	command.Flags().StringVar(&options.Orientation, "orientation", "landscape", "paper orientation: landscape or portrait")
	command.Flags().IntVar(&options.GridColumns, "grid-columns", 1, "automatic grid columns: 1, 2, 3, 4, 6, or 12")
	command.Flags().IntVar(&options.GridGap, "grid-gap", 16, "grid column gap in pixels")
	command.Flags().BoolVar(&options.FailOnWarning, "fail-on-warning", false, "treat conversion warnings as errors")
	command.Flags().BoolVar(&options.Overwrite, "overwrite", false, "replace an existing regular output file")
	_ = command.MarkFlagRequired("output")
	return command
}

func printDiagnostics(writer io.Writer, values []commonentity.Diagnostic) {
	for _, value := range values {
		_, _ = fmt.Fprintf(writer, "%s: %s\n", value.Severity, value.Message())
	}
}
