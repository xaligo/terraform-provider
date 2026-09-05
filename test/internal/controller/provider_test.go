package controller_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	providerschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	application "github.com/xaligo/terraform-provider/internal"
	"github.com/xaligo/terraform-provider/internal/controller"
	commonentity "github.com/xaligo/terraform-provider/internal/entity/common"
	xaligoentity "github.com/xaligo/terraform-provider/internal/entity/xaligo"
)

func TestProviderAndResourceSchemasAreValid(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	instance := application.Provider("test")()
	var metadata frameworkprovider.MetadataResponse
	instance.Metadata(ctx, frameworkprovider.MetadataRequest{}, &metadata)
	if metadata.TypeName != "xaligo" || metadata.Version != "test" {
		t.Fatalf("provider metadata = %#v", metadata)
	}

	var providerSchema frameworkprovider.SchemaResponse
	instance.Schema(ctx, frameworkprovider.SchemaRequest{}, &providerSchema)
	if diagnostics := providerSchema.Schema.ValidateImplementation(ctx); diagnostics.HasError() {
		t.Fatalf("provider schema validation diagnostics = %#v", diagnostics)
	}
	export, ok := providerSchema.Schema.Attributes["export"].(providerschema.StringAttribute)
	if !ok || !export.Optional || export.Required {
		t.Fatalf("provider export schema = %#v", providerSchema.Schema.Attributes["export"])
	}

	diagram := controller.NewDiagramResourceController()
	var resourceMetadata resource.MetadataResponse
	diagram.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "xaligo"}, &resourceMetadata)
	if resourceMetadata.TypeName != "xaligo_diagram" {
		t.Fatalf("resource type name = %q", resourceMetadata.TypeName)
	}
	var resourceSchema resource.SchemaResponse
	diagram.Schema(ctx, resource.SchemaRequest{}, &resourceSchema)
	if diagnostics := resourceSchema.Schema.ValidateImplementation(ctx); diagnostics.HasError() {
		t.Fatalf("resource schema validation diagnostics = %#v", diagnostics)
	}
	frameID, ok := resourceSchema.Schema.Attributes["frame_id"].(resourceschema.StringAttribute)
	if !ok || !frameID.Optional || !frameID.Computed || frameID.Default == nil {
		t.Fatalf("frame_id schema = %#v", resourceSchema.Schema.Attributes["frame_id"])
	}
	for _, name := range []string{"paper_size", "orientation"} {
		attribute, ok := resourceSchema.Schema.Attributes[name].(resourceschema.StringAttribute)
		if !ok || !attribute.Optional || !attribute.Computed || attribute.Default == nil {
			t.Fatalf("%s schema = %#v", name, resourceSchema.Schema.Attributes[name])
		}
	}
	for _, name := range []string{"grid_columns", "grid_gap"} {
		attribute, ok := resourceSchema.Schema.Attributes[name].(resourceschema.Int64Attribute)
		if !ok || !attribute.Optional || !attribute.Computed || attribute.Default == nil {
			t.Fatalf("%s schema = %#v", name, resourceSchema.Schema.Attributes[name])
		}
	}
	row, ok := resourceSchema.Schema.Blocks["row"].(resourceschema.ListNestedBlock)
	if !ok {
		t.Fatalf("row block schema = %#v", resourceSchema.Schema.Blocks["row"])
	}
	items, ok := row.NestedObject.Attributes["items"].(resourceschema.ListAttribute)
	if !ok || !items.Optional || items.ElementType != types.StringType {
		t.Fatalf("row.items schema = %#v", row.NestedObject.Attributes["items"])
	}
	if _, ok := row.NestedObject.Blocks["col"].(resourceschema.ListNestedBlock); !ok {
		t.Fatalf("row.col block schema = %#v", row.NestedObject.Blocks["col"])
	}
	for _, name := range []string{"container", "layout"} {
		if _, ok := resourceSchema.Schema.Blocks[name].(resourceschema.ListNestedBlock); !ok {
			t.Fatalf("%s block schema = %#v", name, resourceSchema.Schema.Blocks[name])
		}
	}
	dataSources := instance.DataSources(ctx)
	if len(dataSources) != 1 {
		t.Fatalf("data source factories = %d, want 1", len(dataSources))
	}
	itemsDataSource := dataSources[0]()
	var dataSourceMetadata datasource.MetadataResponse
	itemsDataSource.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "xaligo"}, &dataSourceMetadata)
	if dataSourceMetadata.TypeName != "xaligo_items" {
		t.Fatalf("items data source type = %q", dataSourceMetadata.TypeName)
	}
	var dataSourceSchema datasource.SchemaResponse
	itemsDataSource.Schema(ctx, datasource.SchemaRequest{}, &dataSourceSchema)
	if diagnostics := dataSourceSchema.Schema.ValidateImplementation(ctx); diagnostics.HasError() {
		t.Fatalf("items data source schema diagnostics = %#v", diagnostics)
	}
	itemsAttribute, ok := dataSourceSchema.Schema.Attributes["items"].(datasourceschema.MapAttribute)
	if !ok || !itemsAttribute.Computed || itemsAttribute.ElementType != types.StringType {
		t.Fatalf("items data source map schema = %#v", dataSourceSchema.Schema.Attributes["items"])
	}
}

func TestProviderConfigurationModes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tests := []struct {
		name       string
		value      any
		wantExport commonentity.ExportMode
		wantError  bool
	}{
		{name: "default", value: nil, wantExport: commonentity.ExportDisable},
		{name: "enable", value: "enable", wantExport: commonentity.ExportEnable},
		{name: "disable", value: "disable", wantExport: commonentity.ExportDisable},
		{name: "invalid", value: "enabled", wantError: true},
		{name: "unknown", value: tftypes.UnknownValue, wantError: true},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			instance := application.Provider("test")()
			var schemaResponse frameworkprovider.SchemaResponse
			instance.Schema(ctx, frameworkprovider.SchemaRequest{}, &schemaResponse)
			providerConfig := providerConfig(schemaResponse.Schema, test.value)

			var validateResponse frameworkprovider.ValidateConfigResponse
			instance.(frameworkprovider.ProviderWithValidateConfig).ValidateConfig(ctx, frameworkprovider.ValidateConfigRequest{Config: providerConfig}, &validateResponse)
			var configureResponse frameworkprovider.ConfigureResponse
			instance.Configure(ctx, frameworkprovider.ConfigureRequest{Config: providerConfig}, &configureResponse)

			if got := validateResponse.Diagnostics.HasError() || configureResponse.Diagnostics.HasError(); got != test.wantError {
				t.Fatalf("configuration diagnostics: validate=%#v configure=%#v", validateResponse.Diagnostics, configureResponse.Diagnostics)
			}
			if test.wantError {
				return
			}
			data, ok := configureResponse.ResourceData.(*xaligoentity.ProviderData)
			if !ok || data.Export != test.wantExport || data.Diagrams == nil {
				t.Fatalf("configured provider data = %#v", configureResponse.ResourceData)
			}
		})
	}
}

func providerConfig(schema providerschema.Schema, value any) tfsdk.Config {
	return tfsdk.Config{
		Raw: tftypes.NewValue(
			tftypes.Object{AttributeTypes: map[string]tftypes.Type{"export": tftypes.String}},
			map[string]tftypes.Value{"export": tftypes.NewValue(tftypes.String, value)},
		),
		Schema: schema,
	}
}
