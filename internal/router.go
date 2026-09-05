package internal

import (
	"context"
	"io"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/spf13/cobra"

	"github.com/xaligo/terraform-provider/internal/controller"
	"github.com/xaligo/terraform-provider/internal/repository"
	"github.com/xaligo/terraform-provider/internal/usecase"
)

// Router dispatches one executable invocation after all dependencies are wired.
type Router func(context.Context, []string) error

func (rcvr Router) Run(ctx context.Context, arguments []string) error {
	return rcvr(ctx, arguments)
}

// InitRouter initializes the combined CLI/provider invocation router.
func InitRouter(version string, stdout, stderr io.Writer) Router {
	providerRouter := InitProviderRouter(version)
	cliRouter := initCLIRouter(version, stdout, stderr, providerRouter)
	return func(ctx context.Context, arguments []string) error {
		if controller.IsCLIInvocation(arguments) {
			return executeCLI(ctx, cliRouter, arguments)
		}
		return providerRouter.Run(ctx, arguments)
	}
}

// InitCLIRouter initializes the Cobra root and registers concept-specific commands.
func InitCLIRouter(version string, stdout, stderr io.Writer) *cobra.Command {
	return initCLIRouter(version, stdout, stderr, InitProviderRouter(version))
}

func initCLIRouter(version string, stdout, stderr io.Writer, providerRouter Router) *cobra.Command {
	return newController(version, stdout, stderr, func(command *cobra.Command, debug bool) error {
		arguments := []string(nil)
		if debug {
			arguments = []string{"--debug"}
		}
		return providerRouter.Run(command.Context(), arguments)
	})
}

// InitProviderRouter initializes provider serving and argument parsing.
func InitProviderRouter(version string) Router {
	server := controller.NewProviderServerController()
	providerFactory := Provider(version)
	return func(ctx context.Context, arguments []string) error {
		debug, err := server.ParseDebug(arguments)
		if err != nil {
			return err
		}
		return server.Serve(ctx, providerFactory, debug)
	}
}

// Provider returns a fresh Terraform provider for each protocol server request.
func Provider(version string) func() frameworkprovider.Provider {
	return func() frameworkprovider.Provider {
		diagrams := newUsecase()
		return controller.NewProviderController(version, diagrams,
			[]func() resource.Resource{func() resource.Resource { return controller.NewDiagramResourceController() }},
			[]func() datasource.DataSource{func() datasource.DataSource { return controller.NewItemsDataSourceController() }},
		)
	}
}

func newRepository() (
	repository.TerraformRepository,
	repository.AWSRepository,
	repository.XaligoRepository,
	repository.PathRepository,
	repository.ArtifactRepository,
) {
	return repository.NewTerraformRepository(),
		repository.NewAWSRepository(),
		repository.NewXaligoRepository(),
		repository.NewPathRepository(),
		repository.NewArtifactRepository()
}

func newUsecase() usecase.DiagramUsecase {
	sources, mapper, marshaler, paths, artifacts := newRepository()
	return usecase.NewDiagramUsecase(sources, mapper, marshaler, paths, artifacts)
}

func newController(version string, stdout, stderr io.Writer, serve controller.ServeFunc) *cobra.Command {
	root := &cobra.Command{
		Use:           "terraform-provider-xaligo",
		Short:         "Terraform provider and Terraform-to-XAL converter",
		SilenceErrors: true,
		SilenceUsage:  true,
		Version:       version,
		Long: `terraform-provider-xaligo serves the xaligo Terraform provider when
started by Terraform. Explicit subcommands expose the same deterministic
Terraform-to-XAL conversion pipeline for local development and automation.`,
	}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetVersionTemplate("{{.Version}}\n")

	commonController := controller.NewCommonController(version, serve)
	converterController := controller.NewConverterController(newUsecase())
	root.AddCommand(converterController.Command())
	root.AddCommand(commonController.VersionCommand())
	root.AddCommand(commonController.ServeCommand())
	return root
}

func executeCLI(ctx context.Context, command *cobra.Command, arguments []string) error {
	command.SetArgs(arguments)
	return command.ExecuteContext(ctx)
}
