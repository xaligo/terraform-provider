package controller

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	providerschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	commonentity "github.com/xaligo/terraform-provider/internal/entity/common"
	xaligoentity "github.com/xaligo/terraform-provider/internal/entity/xaligo"
	"github.com/xaligo/terraform-provider/internal/usecase"
)

const ProviderTypeName = "xaligo"

type ProviderController interface {
	frameworkprovider.Provider
	frameworkprovider.ProviderWithValidateConfig
}

type providerController struct {
	version     string
	diagrams    usecase.DiagramUsecase
	resources   []func() resource.Resource
	dataSources []func() datasource.DataSource
}

var (
	_ frameworkprovider.Provider                   = &providerController{}
	_ frameworkprovider.ProviderWithValidateConfig = &providerController{}
)

func NewProviderController(version string, diagrams usecase.DiagramUsecase, resources []func() resource.Resource, dataSources []func() datasource.DataSource) ProviderController {
	return &providerController{
		version:     version,
		diagrams:    diagrams,
		resources:   append([]func() resource.Resource(nil), resources...),
		dataSources: append([]func() datasource.DataSource(nil), dataSources...),
	}
}

func (rcvr *providerController) Metadata(_ context.Context, _ frameworkprovider.MetadataRequest, response *frameworkprovider.MetadataResponse) {
	response.TypeName = ProviderTypeName
	response.Version = rcvr.version
}

func (rcvr *providerController) Schema(_ context.Context, _ frameworkprovider.SchemaRequest, response *frameworkprovider.SchemaResponse) {
	response.Schema = providerschema.Schema{
		Description: "Generate deterministic xaligo .xal source from Terraform configuration.",
		Attributes: map[string]providerschema.Attribute{
			"export": providerschema.StringAttribute{
				Optional:    true,
				Description: "Generation gate. Valid values are enable and disable; the default is disable.",
			},
		},
	}
}

func (rcvr *providerController) ValidateConfig(ctx context.Context, request frameworkprovider.ValidateConfigRequest, response *frameworkprovider.ValidateConfigResponse) {
	var config xaligoentity.ProviderModel
	response.Diagnostics.Append(request.Config.Get(ctx, &config)...)
	if response.Diagnostics.HasError() || config.Export.IsNull() {
		return
	}
	if config.Export.IsUnknown() {
		response.Diagnostics.AddError(
			"Unknown xaligo export mode",
			"The provider export value must be known and set to either \"enable\" or \"disable\".",
		)
		return
	}
	if !commonentity.ExportMode(config.Export.ValueString()).Valid() {
		response.Diagnostics.AddError(
			"Invalid xaligo export mode",
			fmt.Sprintf("Expected \"enable\" or \"disable\", got %q.", config.Export.ValueString()),
		)
	}
}

func (rcvr *providerController) Configure(ctx context.Context, request frameworkprovider.ConfigureRequest, response *frameworkprovider.ConfigureResponse) {
	var config xaligoentity.ProviderModel
	response.Diagnostics.Append(request.Config.Get(ctx, &config)...)
	if response.Diagnostics.HasError() {
		return
	}
	export := commonentity.ExportDisable
	if !config.Export.IsNull() {
		if config.Export.IsUnknown() {
			response.Diagnostics.AddError(
				"Unknown xaligo export mode",
				"The provider export value must be known during provider configuration.",
			)
			return
		}
		export = commonentity.ExportMode(config.Export.ValueString())
	}
	if !export.Valid() {
		response.Diagnostics.AddError(
			"Invalid xaligo export mode",
			fmt.Sprintf("Expected \"enable\" or \"disable\", got %q.", export),
		)
		return
	}
	if rcvr.diagrams == nil {
		response.Diagnostics.AddError("Provider dependencies are not configured", "The diagram use case is missing.")
		return
	}
	data := &xaligoentity.ProviderData{Export: export, Diagrams: rcvr.diagrams, Items: rcvr.diagrams}
	response.ResourceData = data
	response.DataSourceData = data
}

func (rcvr *providerController) Resources(_ context.Context) []func() resource.Resource {
	return append([]func() resource.Resource(nil), rcvr.resources...)
}

func (rcvr *providerController) DataSources(_ context.Context) []func() datasource.DataSource {
	return append([]func() datasource.DataSource(nil), rcvr.dataSources...)
}
