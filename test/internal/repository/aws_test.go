package repository_test

import (
	"reflect"
	"testing"

	awsentity "github.com/xaligo/terraform-provider/internal/entity/aws"
	commonentity "github.com/xaligo/terraform-provider/internal/entity/common"
	xaligoentity "github.com/xaligo/terraform-provider/internal/entity/xaligo"
	"github.com/xaligo/terraform-provider/internal/repository"
)

func TestMapUsesAWSCloudRegionAndPublicSubnetTags(t *testing.T) {
	t.Parallel()

	graph, graphDiagnostics := commonentity.BuildGraph(commonentity.TerraformConfig{Blocks: []commonentity.Block{
		{
			Kind: commonentity.BlockResource, Type: "aws_vpc", Name: "main", Address: "aws_vpc.main",
		},
		{
			Kind: commonentity.BlockResource, Type: "aws_subnet", Name: "public", Address: "aws_subnet.public",
			Attributes: map[string]commonentity.Attribute{
				"vpc_id":            referenceAttribute("vpc_id", "aws_vpc.main"),
				"availability_zone": stringAttribute("availability_zone", "ap-northeast-1a"),
			},
			References: []string{"aws_vpc.main"},
		},
		{
			Kind: commonentity.BlockResource, Type: "aws_internet_gateway", Name: "main", Address: "aws_internet_gateway.main",
			Attributes: map[string]commonentity.Attribute{
				"vpc_id": referenceAttribute("vpc_id", "aws_vpc.main"),
			},
			References: []string{"aws_vpc.main"},
		},
		{
			Kind: commonentity.BlockResource, Type: "aws_route_table", Name: "public", Address: "aws_route_table.public",
			Attributes: map[string]commonentity.Attribute{
				"vpc_id": referenceAttribute("vpc_id", "aws_vpc.main"),
			},
			References: []string{"aws_vpc.main", "aws_internet_gateway.main"},
		},
		{
			Kind: commonentity.BlockResource, Type: "aws_security_group", Name: "web", Address: "aws_security_group.web",
			Attributes: map[string]commonentity.Attribute{
				"vpc_id": referenceAttribute("vpc_id", "aws_vpc.main"),
			},
			References: []string{"aws_vpc.main"},
		},
		{
			Kind: commonentity.BlockResource, Type: "aws_route_table_association", Name: "public", Address: "aws_route_table_association.public",
			Attributes: map[string]commonentity.Attribute{
				"subnet_id":      referenceAttribute("subnet_id", "aws_subnet.public"),
				"route_table_id": referenceAttribute("route_table_id", "aws_route_table.public"),
			},
			References: []string{"aws_subnet.public", "aws_route_table.public"},
		},
		{
			Kind: commonentity.BlockResource, Type: "aws_instance", Name: "web", Address: "aws_instance.web",
			Attributes: map[string]commonentity.Attribute{
				"subnet_id": referenceAttribute("subnet_id", "aws_subnet.public"),
			},
			References: []string{"aws_subnet.public"},
		},
		{Kind: commonentity.BlockResource, Type: "aws_s3_bucket", Name: "logs", Address: "aws_s3_bucket.logs"},
		{Kind: commonentity.BlockResource, Type: "aws_iam_role", Name: "worker", Address: "aws_iam_role.worker"},
	}})
	if len(graphDiagnostics) != 0 {
		t.Fatalf("BuildGraph() diagnostics = %#v", graphDiagnostics)
	}

	document, diagnostics := repository.NewAWSRepository().Map(graph, xaligoentity.DiagramOptions{FrameID: "main"})
	if commonentity.HasErrors(diagnostics) {
		t.Fatalf("Map() diagnostics = %#v", diagnostics)
	}
	if hasDiagnosticCode(diagnostics, "MAPPING-W001") {
		t.Fatalf("proven public subnet retained neutral warning: %#v", diagnostics)
	}
	cloud := requireElement(t, document.Frame.Children, "aws-cloud", "xaligo-aws-cloud")
	global := requireElement(t, cloud.Children, "generic-group", "xaligo-aws-global-services")
	requireElement(t, global.Children, "item", "1479")
	region := requireElement(t, cloud.Children, "region", "xaligo-aws-region-ap-northeast-1")
	regional := requireElement(t, region.Children, "generic-group", "xaligo-aws-region-ap-northeast-1-services")
	requireElement(t, regional.Children, "item", "1642")
	vpc := requireElement(t, region.Children, "vpc", "aws_vpc-main")
	requireElement(t, vpc.Children, "security-group", "aws_security_group-web")
	requireElement(t, vpc.Children, "rectangle", "aws_route_table-public")
	availabilityZone := requireElement(t, vpc.Children, "availability-zone", "aws_vpc-main-availability-zone-ap-northeast-1a")
	publicSubnet := requireElement(t, availabilityZone.Children, "public-subnet", "aws_subnet-public")
	requireElement(t, publicSubnet.Children, "item", "1790")
}

func TestMapKeepsSubnetNeutralWhenInternetRouteIsNotProven(t *testing.T) {
	t.Parallel()

	graph, _ := commonentity.BuildGraph(commonentity.TerraformConfig{Blocks: []commonentity.Block{
		{Kind: commonentity.BlockResource, Type: "aws_vpc", Name: "main", Address: "aws_vpc.main"},
		{
			Kind: commonentity.BlockResource, Type: "aws_subnet", Name: "unknown", Address: "aws_subnet.unknown",
			Attributes: map[string]commonentity.Attribute{
				"vpc_id":            referenceAttribute("vpc_id", "aws_vpc.main"),
				"availability_zone": stringAttribute("availability_zone", "ap-northeast-1a"),
			},
			References: []string{"aws_vpc.main"},
		},
	}})
	document, diagnostics := repository.NewAWSRepository().Map(graph, xaligoentity.DiagramOptions{FrameID: "main"})
	if !hasDiagnosticCode(diagnostics, "MAPPING-W001") {
		t.Fatalf("Map() diagnostics = %#v, want MAPPING-W001", diagnostics)
	}
	requireElement(t, document.Frame.Children, "generic-group", "aws_subnet-unknown")
}

func TestMapUsesPublicSubnetForStandaloneRoute(t *testing.T) {
	t.Parallel()

	graph, _ := commonentity.BuildGraph(commonentity.TerraformConfig{Blocks: []commonentity.Block{
		{Kind: commonentity.BlockResource, Type: "aws_vpc", Name: "main", Address: "aws_vpc.main"},
		{
			Kind: commonentity.BlockResource, Type: "aws_subnet", Name: "public", Address: "aws_subnet.public",
			Attributes: map[string]commonentity.Attribute{
				"vpc_id":            referenceAttribute("vpc_id", "aws_vpc.main"),
				"availability_zone": stringAttribute("availability_zone", "ap-northeast-1a"),
			},
			References: []string{"aws_vpc.main"},
		},
		{
			Kind: commonentity.BlockResource, Type: "aws_internet_gateway", Name: "main", Address: "aws_internet_gateway.main",
			Attributes: map[string]commonentity.Attribute{
				"vpc_id": referenceAttribute("vpc_id", "aws_vpc.main"),
			},
			References: []string{"aws_vpc.main"},
		},
		{
			Kind: commonentity.BlockResource, Type: "aws_route_table", Name: "public", Address: "aws_route_table.public",
			Attributes: map[string]commonentity.Attribute{
				"vpc_id": referenceAttribute("vpc_id", "aws_vpc.main"),
			},
			References: []string{"aws_vpc.main"},
		},
		{
			Kind: commonentity.BlockResource, Type: "aws_route", Name: "internet", Address: "aws_route.internet",
			Attributes: map[string]commonentity.Attribute{
				"route_table_id": referenceAttribute("route_table_id", "aws_route_table.public"),
				"gateway_id":     referenceAttribute("gateway_id", "aws_internet_gateway.main"),
			},
			References: []string{"aws_route_table.public", "aws_internet_gateway.main"},
		},
		{
			Kind: commonentity.BlockResource, Type: "aws_route_table_association", Name: "public", Address: "aws_route_table_association.public",
			Attributes: map[string]commonentity.Attribute{
				"subnet_id":      referenceAttribute("subnet_id", "aws_subnet.public"),
				"route_table_id": referenceAttribute("route_table_id", "aws_route_table.public"),
			},
			References: []string{"aws_subnet.public", "aws_route_table.public"},
		},
	}})
	document, diagnostics := repository.NewAWSRepository().Map(graph, xaligoentity.DiagramOptions{FrameID: "main"})
	if commonentity.HasErrors(diagnostics) || hasDiagnosticCode(diagnostics, "MAPPING-W001") {
		t.Fatalf("Map() diagnostics = %#v", diagnostics)
	}
	requireElement(t, document.Frame.Children, "public-subnet", "aws_subnet-public")
}

func TestMapUsesPrivateSubnetForProvenNATRoute(t *testing.T) {
	t.Parallel()

	graph, _ := commonentity.BuildGraph(commonentity.TerraformConfig{Blocks: []commonentity.Block{
		{Kind: commonentity.BlockResource, Type: "aws_vpc", Name: "main", Address: "aws_vpc.main"},
		{
			Kind: commonentity.BlockResource, Type: "aws_subnet", Name: "egress", Address: "aws_subnet.egress",
			Attributes: map[string]commonentity.Attribute{
				"vpc_id":            referenceAttribute("vpc_id", "aws_vpc.main"),
				"availability_zone": stringAttribute("availability_zone", "ap-northeast-1a"),
			},
			References: []string{"aws_vpc.main"},
		},
		{
			Kind: commonentity.BlockResource, Type: "aws_subnet", Name: "private", Address: "aws_subnet.private",
			Attributes: map[string]commonentity.Attribute{
				"vpc_id":            referenceAttribute("vpc_id", "aws_vpc.main"),
				"availability_zone": stringAttribute("availability_zone", "ap-northeast-1a"),
			},
			References: []string{"aws_vpc.main"},
		},
		{
			Kind: commonentity.BlockResource, Type: "aws_nat_gateway", Name: "main", Address: "aws_nat_gateway.main",
			Attributes: map[string]commonentity.Attribute{
				"subnet_id": referenceAttribute("subnet_id", "aws_subnet.egress"),
			},
			References: []string{"aws_subnet.egress"},
		},
		{
			Kind: commonentity.BlockResource, Type: "aws_route_table", Name: "private", Address: "aws_route_table.private",
			Attributes: map[string]commonentity.Attribute{
				"vpc_id": referenceAttribute("vpc_id", "aws_vpc.main"),
			},
			References: []string{"aws_vpc.main", "aws_nat_gateway.main"},
		},
		{
			Kind: commonentity.BlockResource, Type: "aws_route_table_association", Name: "private", Address: "aws_route_table_association.private",
			Attributes: map[string]commonentity.Attribute{
				"subnet_id":      referenceAttribute("subnet_id", "aws_subnet.private"),
				"route_table_id": referenceAttribute("route_table_id", "aws_route_table.private"),
			},
			References: []string{"aws_subnet.private", "aws_route_table.private"},
		},
	}})
	document, diagnostics := repository.NewAWSRepository().Map(graph, xaligoentity.DiagramOptions{FrameID: "main"})
	if commonentity.HasErrors(diagnostics) {
		t.Fatalf("Map() diagnostics = %#v", diagnostics)
	}
	requireElement(t, document.Frame.Children, "private-subnet", "aws_subnet-private")
}

func TestMapUsesReviewedCatalogAndGenericFallback(t *testing.T) {
	t.Parallel()

	graph, graphDiagnostics := commonentity.BuildGraph(commonentity.TerraformConfig{Blocks: []commonentity.Block{
		{Kind: commonentity.BlockResource, Type: "custom_service", Name: "worker", Address: "custom_service.worker"},
		{Kind: commonentity.BlockResource, Type: "aws_s3_bucket", Name: "logs", Address: "aws_s3_bucket.logs"},
	}})
	if len(graphDiagnostics) != 0 {
		t.Fatalf("BuildGraph() diagnostics = %#v", graphDiagnostics)
	}
	document, diagnostics := repository.NewAWSRepository().Map(graph, xaligoentity.DiagramOptions{FrameID: "main", Title: "Test"})
	if commonentity.HasErrors(diagnostics) {
		t.Fatalf("Map() diagnostics = %#v", diagnostics)
	}
	if got, want := diagnosticCodeList(diagnostics), []string{"MAPPING-W002"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("diagnostic codes = %v, want %v", got, want)
	}
	cloud := requireElement(t, document.Frame.Children, "aws-cloud", "xaligo-aws-cloud")
	s3 := requireElement(t, cloud.Children, "item", "1642")
	if s3.Name != "aws_s3_bucket-logs" {
		t.Errorf("S3 mapping = %#v", s3)
	}
	terraformResources := requireElement(t, document.Frame.Children, "generic-group", "xaligo-terraform-resources")
	fallback := requireElement(t, terraformResources.Children, "rectangle", "custom_service-worker")
	if fallback.Title != "custom_service.worker" {
		t.Errorf("fallback mapping = %#v", fallback)
	}
}

func TestMapAppliesExplicitRowFromGlobalItemsMap(t *testing.T) {
	t.Parallel()

	graph, _ := commonentity.BuildGraph(commonentity.TerraformConfig{Blocks: []commonentity.Block{
		{Kind: commonentity.BlockResource, Type: "aws_s3_bucket", Name: "logs", Address: "aws_s3_bucket.logs"},
		{Kind: commonentity.BlockResource, Type: "aws_lambda_function", Name: "worker", Address: "aws_lambda_function.worker"},
		{Kind: commonentity.BlockResource, Type: "aws_sqs_queue", Name: "jobs", Address: "aws_sqs_queue.jobs"},
	}})
	document, diagnostics := repository.NewAWSRepository().Map(graph, xaligoentity.DiagramOptions{
		FrameID: "main", GridGap: 24, Rows: []commonentity.RowSpec{{Items: []string{
			"items.aws_s3_bucket.logs", "items.aws_lambda_function.worker", "items.aws_sqs_queue.jobs",
		}}},
	})
	if commonentity.HasErrors(diagnostics) {
		t.Fatalf("Map() diagnostics = %#v", diagnostics)
	}
	cloud := requireElement(t, document.Frame.Children, "aws-cloud", "xaligo-aws-cloud")
	region := requireElement(t, cloud.Children, "region", "xaligo-aws-region-unspecified")
	services := requireElement(t, region.Children, "generic-group", "xaligo-aws-region-unspecified-services")
	if len(services.Children) != 1 || services.Children[0].Tag != "row" || services.Children[0].Gap != 24 {
		t.Fatalf("explicit services row = %#v", services.Children)
	}
	if cols := services.Children[0].Children; len(cols) != 3 || cols[0].Span != 4 {
		t.Fatalf("explicit services columns = %#v", cols)
	}
}

func TestMapRejectsNormalizedIdentityCollisions(t *testing.T) {
	t.Parallel()

	graph, _ := commonentity.BuildGraph(commonentity.TerraformConfig{Blocks: []commonentity.Block{
		{Kind: commonentity.BlockResource, Type: "test-a", Name: "b", Address: "test-a.b"},
		{Kind: commonentity.BlockResource, Type: "test", Name: "a-b", Address: "test.a-b"},
	}})
	_, diagnostics := repository.NewAWSRepository().Map(graph, xaligoentity.DiagramOptions{FrameID: "main"})
	if !commonentity.HasErrors(diagnostics) || diagnostics[0].Code != "MAPPING-E002" {
		t.Fatalf("Map() diagnostics = %#v", diagnostics)
	}
}

func TestLoadBalancerCatalogSelection(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"application": "1592",
		"gateway":     "1594",
		"network":     "1595",
		"unknown":     "1182",
	}
	for loadBalancerType, want := range tests {
		loadBalancerType, want := loadBalancerType, want
		t.Run(loadBalancerType, func(t *testing.T) {
			t.Parallel()
			mapping, ok := awsentity.LookupItem("aws_lb", loadBalancerType)
			if !ok || mapping.CatalogID != want {
				t.Fatalf("mappingFor(%q) = %#v, %v; want catalog %s", loadBalancerType, mapping, ok, want)
			}
		})
	}
}

func TestNormalizeIdentifier(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"aws_vpc.main":          "aws_vpc-main",
		"  AWS::Service / Name": "aws-service-name",
		"---":                   "",
	}
	for input, want := range tests {
		if got := repository.NormalizeIdentifier(input); got != want {
			t.Errorf("NormalizeIdentifier(%q) = %q, want %q", input, got, want)
		}
	}
}

func diagnosticCodeList(values []commonentity.Diagnostic) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.Code)
	}
	return result
}

func stringAttribute(name, value string) commonentity.Attribute {
	return commonentity.Attribute{
		Name:  name,
		Value: commonentity.Value{Kind: commonentity.ValueString, String: value},
	}
}

func referenceAttribute(name string, references ...string) commonentity.Attribute {
	return commonentity.Attribute{Name: name, References: references}
}

func hasDiagnosticCode(diagnostics []commonentity.Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func requireElement(t *testing.T, elements []xaligoentity.XALElement, tag, id string) xaligoentity.XALElement {
	t.Helper()
	for _, element := range elements {
		if element.Tag == tag && element.ID == id {
			return element
		}
		if nested, found := findElement(element.Children, tag, id); found {
			return nested
		}
	}
	t.Fatalf("element <%s id=%q> not found in %#v", tag, id, elements)
	return xaligoentity.XALElement{}
}

func findElement(elements []xaligoentity.XALElement, tag, id string) (xaligoentity.XALElement, bool) {
	for _, element := range elements {
		if element.Tag == tag && element.ID == id {
			return element, true
		}
		if nested, found := findElement(element.Children, tag, id); found {
			return nested, true
		}
	}
	return xaligoentity.XALElement{}, false
}
