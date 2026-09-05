package controller

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	xaligoentity "github.com/xaligo/terraform-provider/internal/entity/xaligo"
)

type ItemsDataSourceController interface {
	datasource.DataSource
	datasource.DataSourceWithConfigure
}

type itemsDataSourceController struct {
	providerData *xaligoentity.ProviderData
}

func NewItemsDataSourceController() ItemsDataSourceController {
	return &itemsDataSourceController{}
}

func (rcvr *itemsDataSourceController) Metadata(_ context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_items"
}

func (rcvr *itemsDataSourceController) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Description: "Read every Terraform resource, data source, and module address into one layout items map.",
		Attributes: map[string]schema.Attribute{
			"source_dir": schema.StringAttribute{Required: true, Description: "Directory containing direct regular .tf inputs."},
			"items":      schema.MapAttribute{Computed: true, ElementType: types.StringType, Description: "Terraform address map used by xaligo layout blocks."},
		},
	}
}

func (rcvr *itemsDataSourceController) Configure(_ context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}
	data, ok := request.ProviderData.(*xaligoentity.ProviderData)
	if !ok {
		response.Diagnostics.AddError("Unexpected provider configuration type", fmt.Sprintf("Expected xaligo provider data, got %T.", request.ProviderData))
		return
	}
	rcvr.providerData = data
}

func (rcvr *itemsDataSourceController) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	if rcvr.providerData == nil || rcvr.providerData.Items == nil {
		response.Diagnostics.AddError("Provider is not configured", "The xaligo provider did not supply the items service.")
		return
	}
	var config xaligoentity.ItemsDataSourceModel
	response.Diagnostics.Append(request.Config.Get(ctx, &config)...)
	if response.Diagnostics.HasError() || config.SourceDir.IsNull() || config.SourceDir.IsUnknown() {
		return
	}
	items, diagnostics, err := rcvr.providerData.Items.Items(ctx, config.SourceDir.ValueString())
	appendDomainDiagnostics(&response.Diagnostics, diagnostics)
	if err != nil {
		response.Diagnostics.AddError("Terraform items could not be loaded", err.Error())
		return
	}
	value, mapDiagnostics := types.MapValueFrom(ctx, types.StringType, items)
	response.Diagnostics.Append(mapDiagnostics...)
	if response.Diagnostics.HasError() {
		return
	}
	config.Items = value
	response.Diagnostics.Append(response.State.Set(ctx, &config)...)
}
