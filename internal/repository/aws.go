package repository

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	awsentity "github.com/xaligo/terraform-provider/internal/entity/aws"
	commonentity "github.com/xaligo/terraform-provider/internal/entity/common"
	xaligoentity "github.com/xaligo/terraform-provider/internal/entity/xaligo"
)

type AWSRepository interface {
	Map(graph commonentity.Graph, options xaligoentity.DiagramOptions) (xaligoentity.XALDocument, []commonentity.Diagnostic)
}

type awsRepository struct {
	graph          commonentity.Graph
	identities     map[string]string
	identityOwners map[string]string
	consumed       map[string]bool
	usedReferences map[string]map[string]bool
	diagnostics    []commonentity.Diagnostic
}

func NewAWSRepository() AWSRepository {
	return &awsRepository{}
}

func (rcvr *awsRepository) Map(graph commonentity.Graph, options xaligoentity.DiagramOptions) (xaligoentity.XALDocument, []commonentity.Diagnostic) {
	session := &awsRepository{}
	session.graph = graph
	session.identities = make(map[string]string, len(graph.Nodes))
	session.identityOwners = map[string]string{options.FrameID: "frame " + options.FrameID}
	session.consumed = make(map[string]bool, len(graph.Nodes))
	session.usedReferences = make(map[string]map[string]bool, len(graph.Nodes))
	session.prepareIdentities()
	if commonentity.HasErrors(session.diagnostics) {
		commonentity.SortDiagnostics(session.diagnostics)
		return xaligoentity.XALDocument{}, session.diagnostics
	}

	vpcsByRegion := session.mapVPCsByRegion()
	global, regional, unresolvedAWS, remaining := session.mapRemainingByScope()
	cloudChildren := session.mapCloudChildren(vpcsByRegion, global, regional, unresolvedAWS)
	var children []xaligoentity.XALElement
	if len(cloudChildren) > 0 && session.reserveIdentity("xaligo-aws-cloud", "generated AWS cloud") {
		children = append(children, xaligoentity.XALElement{
			Tag:      "aws-cloud",
			ID:       "xaligo-aws-cloud",
			Title:    "AWS Cloud",
			Children: cloudChildren,
		})
	}
	if len(remaining) > 0 && session.reserveIdentity("xaligo-terraform-resources", "generated Terraform resources") {
		children = append(children, xaligoentity.XALElement{
			Tag:      "generic-group",
			ID:       "xaligo-terraform-resources",
			Title:    "Terraform resources",
			Children: remaining,
		})
	}
	session.reportOmittedDependencies()
	var layoutError error
	children, layoutError = xaligoentity.ApplyExplicitLayout(children, options.Containers, options.Rows, options.Layouts, options.GridGap)
	if layoutError != nil {
		session.diagnostics = append(session.diagnostics, commonentity.Diagnostic{
			Code: "MAPPING-E005", Severity: commonentity.SeverityError,
			Summary: "Explicit row layout is invalid", Detail: layoutError.Error(),
		})
	}
	commonentity.SortDiagnostics(session.diagnostics)
	if commonentity.HasErrors(session.diagnostics) {
		return xaligoentity.XALDocument{}, session.diagnostics
	}
	return xaligoentity.NewXALDocument(options, children), session.diagnostics
}

func (rcvr *awsRepository) prepareIdentities() {
	for _, node := range rcvr.graph.Nodes {
		identity := NormalizeIdentifier(node.Address)
		if identity == "" {
			rcvr.diagnostics = append(rcvr.diagnostics, commonentity.Diagnostic{
				Code:     "MAPPING-E001",
				Severity: commonentity.SeverityError,
				Summary:  "Terraform address cannot form a XAL identity",
				Detail:   fmt.Sprintf("Address %q has no usable identity characters.", node.Address),
				Range:    node.Range,
			})
			continue
		}
		if previous, exists := rcvr.identityOwners[identity]; exists {
			rcvr.diagnostics = append(rcvr.diagnostics, commonentity.Diagnostic{
				Code:     "MAPPING-E002",
				Severity: commonentity.SeverityError,
				Summary:  "Generated XAL identity collision",
				Detail:   fmt.Sprintf("Terraform addresses %q and %q both normalize to %q.", previous, node.Address, identity),
				Range:    node.Range,
			})
			continue
		}
		rcvr.identities[node.Address] = identity
		rcvr.identityOwners[identity] = node.Address
	}
}

func (rcvr *awsRepository) mapVPCsByRegion() map[string][]xaligoentity.XALElement {
	result := map[string][]xaligoentity.XALElement{}
	for _, node := range rcvr.graph.Nodes {
		if node.Kind != commonentity.NodeResource || node.Type != "aws_vpc" {
			continue
		}
		rcvr.consumed[node.Address] = true
		vpc := xaligoentity.XALElement{Tag: "vpc", ID: rcvr.identities[node.Address], Title: nodeTitle(node)}
		vpc.Children = rcvr.mapVPCChildren(node)
		region := rcvr.regionForVPC(node)
		result[region] = append(result[region], vpc)
	}
	return result
}

func (rcvr *awsRepository) mapVPCChildren(vpc commonentity.Node) []xaligoentity.XALElement {
	var subnets []commonentity.Node
	for _, node := range rcvr.graph.Nodes {
		if node.Kind != commonentity.NodeResource || node.Type != "aws_subnet" || rcvr.consumed[node.Address] {
			continue
		}
		if rcvr.hasUniqueAttributeReference(node, "vpc_id", vpc.Address) {
			subnets = append(subnets, node)
		}
	}

	byAvailabilityZone := map[string][]commonentity.Node{}
	var withoutAvailabilityZone []commonentity.Node
	for _, subnet := range subnets {
		if availabilityZone, known := subnet.String("availability_zone"); known && strings.TrimSpace(availabilityZone) != "" {
			byAvailabilityZone[availabilityZone] = append(byAvailabilityZone[availabilityZone], subnet)
		} else {
			withoutAvailabilityZone = append(withoutAvailabilityZone, subnet)
		}
	}

	var children []xaligoentity.XALElement
	availabilityZones := make([]string, 0, len(byAvailabilityZone))
	for availabilityZone := range byAvailabilityZone {
		availabilityZones = append(availabilityZones, availabilityZone)
	}
	sort.Strings(availabilityZones)
	for _, availabilityZone := range availabilityZones {
		availabilityZoneID := rcvr.identities[vpc.Address] + "-availability-zone-" + NormalizeIdentifier(availabilityZone)
		if previous, exists := rcvr.identityOwners[availabilityZoneID]; exists {
			rcvr.diagnostics = append(rcvr.diagnostics, commonentity.Diagnostic{
				Code:     "MAPPING-E002",
				Severity: commonentity.SeverityError,
				Summary:  "Generated XAL identity collision",
				Detail:   fmt.Sprintf("Availability zone id %q conflicts with %q.", availabilityZoneID, previous),
			})
			continue
		}
		rcvr.identityOwners[availabilityZoneID] = "availability zone " + availabilityZone
		group := xaligoentity.XALElement{Tag: "availability-zone", ID: availabilityZoneID, Title: "AZ: " + availabilityZone}
		for _, subnet := range byAvailabilityZone[availabilityZone] {
			group.Children = append(group.Children, rcvr.mapSubnet(subnet))
		}
		children = append(children, group)
	}
	for _, subnet := range withoutAvailabilityZone {
		children = append(children, rcvr.mapSubnet(subnet))
	}

	var vpcItems []xaligoentity.XALElement
	var vpcGroups []xaligoentity.XALElement
	for _, node := range rcvr.graph.Nodes {
		if node.Kind != commonentity.NodeResource || rcvr.consumed[node.Address] {
			continue
		}
		if group, dedicated := awsentity.LookupGroup(node.Type); dedicated && group.Scope == awsentity.ScopeVPC && rcvr.hasUniqueAttributeReference(node, "vpc_id", vpc.Address) {
			rcvr.consumed[node.Address] = true
			vpcGroups = append(vpcGroups, rcvr.nodeElement(node))
			continue
		}
		mapping, mapped := mappingFor(node)
		if mapped && mapping.Scope == awsentity.ScopeVPC {
			contained := rcvr.hasUniqueAttributeReference(node, "vpc_id", vpc.Address)
			if !contained && node.Type == "aws_lb" {
				contained = rcvr.referencesAnySubnetInVPC(node, vpc.Address)
			}
			if contained {
				rcvr.consumed[node.Address] = true
				vpcItems = append(vpcItems, rcvr.nodeElement(node))
				continue
			}
		}
		if !isAWSNode(node) || !rcvr.hasUniqueAttributeReference(node, "vpc_id", vpc.Address) {
			continue
		}
		rcvr.consumed[node.Address] = true
		vpcItems = append(vpcItems, rcvr.nodeElement(node))
	}
	if len(vpcItems) > 0 {
		groupID := rcvr.identities[vpc.Address] + "-resources"
		if rcvr.reserveIdentity(groupID, "resources for "+vpc.Address) {
			children = append([]xaligoentity.XALElement{{
				Tag: "generic-group", ID: groupID, Title: "VPC resources", Children: vpcItems,
			}}, children...)
		}
	}
	if len(vpcGroups) > 0 {
		children = append(vpcGroups, children...)
	}
	return children
}

func (rcvr *awsRepository) mapSubnet(subnet commonentity.Node) xaligoentity.XALElement {
	rcvr.consumed[subnet.Address] = true
	tag, proven := rcvr.subnetGroupTag(subnet)
	element := xaligoentity.XALElement{
		Tag:   tag,
		ID:    rcvr.identities[subnet.Address],
		Title: "Subnet: " + nodeTitle(subnet),
	}
	if !proven {
		rcvr.diagnostics = append(rcvr.diagnostics, commonentity.Diagnostic{
			Code:     "MAPPING-W001",
			Severity: commonentity.SeverityWarning,
			Summary:  "Subnet visibility is not proven",
			Detail:   fmt.Sprintf("%s is rendered as a neutral group because a public or private route cannot be proven from the initial static mapping.", subnet.Address),
			Range:    subnet.Range,
		})
	}

	for _, node := range rcvr.graph.Nodes {
		if node.Kind != commonentity.NodeResource || rcvr.consumed[node.Address] {
			continue
		}
		if rcvr.hasUniqueAttributeReference(node, "subnet_id", subnet.Address) {
			rcvr.consumed[node.Address] = true
			element.Children = append(element.Children, rcvr.nodeElement(node))
		}
	}
	return element
}

func (rcvr *awsRepository) mapRemainingByScope() (global, regional, unresolvedAWS, remaining []xaligoentity.XALElement) {
	for _, node := range rcvr.graph.Nodes {
		if rcvr.consumed[node.Address] {
			continue
		}
		rcvr.consumed[node.Address] = true
		element := rcvr.nodeElement(node)
		if !isAWSNode(node) {
			remaining = append(remaining, element)
			continue
		}
		if group, mapped := awsentity.LookupGroup(node.Type); mapped && node.Kind == commonentity.NodeResource {
			switch group.Scope {
			case awsentity.ScopeGlobal:
				global = append(global, element)
				continue
			case awsentity.ScopeRegional:
				regional = append(regional, element)
				continue
			}
		}
		mapping, mapped := mappingFor(node)
		if mapped && node.Kind == commonentity.NodeResource {
			switch mapping.Scope {
			case awsentity.ScopeGlobal:
				global = append(global, element)
				continue
			case awsentity.ScopeRegional:
				regional = append(regional, element)
				continue
			}
		}
		unresolvedAWS = append(unresolvedAWS, element)
	}
	return global, regional, unresolvedAWS, remaining
}

func (rcvr *awsRepository) mapCloudChildren(vpcsByRegion map[string][]xaligoentity.XALElement, global, regional, unresolvedAWS []xaligoentity.XALElement) []xaligoentity.XALElement {
	var children []xaligoentity.XALElement
	if len(global) > 0 && rcvr.reserveIdentity("xaligo-aws-global-services", "generated global AWS services") {
		children = append(children, xaligoentity.XALElement{
			Tag: "generic-group", ID: "xaligo-aws-global-services", Title: "Global services", Children: global,
		})
	}

	regionNames := make([]string, 0, len(vpcsByRegion))
	for region := range vpcsByRegion {
		regionNames = append(regionNames, region)
	}
	if len(regional) > 0 && len(regionNames) == 0 {
		vpcsByRegion[""] = nil
		regionNames = append(regionNames, "")
	}
	sort.Strings(regionNames)
	regionalRegion := ""
	placeRegional := len(regionNames) == 1
	if placeRegional {
		regionalRegion = regionNames[0]
	} else if len(regional) > 0 {
		unresolvedAWS = append(unresolvedAWS, regional...)
	}

	for _, region := range regionNames {
		regionID := awsRegionID(region)
		var regionChildren []xaligoentity.XALElement
		if len(regional) > 0 && placeRegional && region == regionalRegion {
			servicesID := regionID + "-services"
			if rcvr.reserveIdentity(servicesID, "generated regional AWS services") {
				regionChildren = append(regionChildren, xaligoentity.XALElement{
					Tag: "generic-group", ID: servicesID, Title: "Regional services", Children: regional,
				})
			}
		}
		regionChildren = append(regionChildren, vpcsByRegion[region]...)
		if !rcvr.reserveIdentity(regionID, "generated AWS region") {
			continue
		}
		title := "AWS Region (unspecified)"
		if region != "" {
			title = "Region: " + region
		}
		children = append(children, xaligoentity.XALElement{
			Tag: "region", ID: regionID, Title: title, Children: regionChildren,
		})
	}

	if len(unresolvedAWS) > 0 && rcvr.reserveIdentity("xaligo-aws-unresolved-resources", "generated unresolved AWS resources") {
		children = append(children, xaligoentity.XALElement{
			Tag: "generic-group", ID: "xaligo-aws-unresolved-resources", Title: "AWS resources (scope unresolved)", Children: unresolvedAWS,
		})
	}
	return children
}

func (rcvr *awsRepository) regionForVPC(vpc commonentity.Node) string {
	regions := map[string]bool{}
	for _, node := range rcvr.graph.Nodes {
		if node.Kind != commonentity.NodeResource || node.Type != "aws_subnet" {
			continue
		}
		if !rcvr.hasUniqueAttributeReference(node, "vpc_id", vpc.Address) {
			continue
		}
		availabilityZone, known := node.String("availability_zone")
		if !known {
			continue
		}
		if region, ok := awsRegionFromAvailabilityZone(availabilityZone); ok {
			regions[region] = true
		}
	}
	if len(regions) != 1 {
		return ""
	}
	for region := range regions {
		return region
	}
	return ""
}

func (rcvr *awsRepository) subnetGroupTag(subnet commonentity.Node) (string, bool) {
	vpcReferences := subnet.AttributeReferences("vpc_id")
	if len(vpcReferences) != 1 {
		return "generic-group", false
	}
	vpcAddress := vpcReferences[0]
	publicRoute := false
	privateRoute := false
	for _, association := range rcvr.graph.Nodes {
		if association.Kind != commonentity.NodeResource || association.Type != "aws_route_table_association" {
			continue
		}
		if !rcvr.hasUniqueAttributeReference(association, "subnet_id", subnet.Address) {
			continue
		}
		routeTableReferences := association.AttributeReferences("route_table_id")
		if len(routeTableReferences) != 1 {
			continue
		}
		routeTable, exists := rcvr.graph.Node(routeTableReferences[0])
		if !exists || routeTable.Type != "aws_route_table" || !rcvr.hasUniqueAttributeReference(routeTable, "vpc_id", vpcAddress) {
			continue
		}
		tablePublic, tablePrivate := rcvr.gatewayRouteKind(routeTable, vpcAddress)
		associationPublic := tablePublic
		associationPrivate := tablePrivate
		for _, route := range rcvr.graph.Nodes {
			if route.Kind != commonentity.NodeResource || route.Type != "aws_route" ||
				!rcvr.hasUniqueAttributeReference(route, "route_table_id", routeTable.Address) {
				continue
			}
			routePublic, routePrivate := rcvr.gatewayRouteKind(route, vpcAddress)
			associationPublic = associationPublic || routePublic
			associationPrivate = associationPrivate || routePrivate
		}
		publicRoute = publicRoute || associationPublic
		privateRoute = privateRoute || associationPrivate
		if associationPublic || associationPrivate {
			rcvr.markReferenceUsed(association.Address, routeTable.Address)
		}
	}
	if publicRoute && !privateRoute {
		return "public-subnet", true
	}
	if privateRoute && !publicRoute {
		return "private-subnet", true
	}
	return "generic-group", false
}

func (rcvr *awsRepository) gatewayRouteKind(route commonentity.Node, vpcAddress string) (public, private bool) {
	for _, dependency := range route.DependsOn {
		gateway, exists := rcvr.graph.Node(dependency)
		if !exists || gateway.Kind != commonentity.NodeResource {
			continue
		}
		switch gateway.Type {
		case "aws_internet_gateway":
			if rcvr.hasUniqueAttributeReference(gateway, "vpc_id", vpcAddress) {
				public = true
				rcvr.markReferenceUsed(route.Address, gateway.Address)
			}
		case "aws_nat_gateway":
			if rcvr.natGatewayBelongsToVPC(gateway, vpcAddress) {
				private = true
				rcvr.markReferenceUsed(route.Address, gateway.Address)
			}
		}
	}
	return public, private
}

func (rcvr *awsRepository) natGatewayBelongsToVPC(gateway commonentity.Node, vpcAddress string) bool {
	subnetReferences := gateway.AttributeReferences("subnet_id")
	if len(subnetReferences) != 1 {
		return false
	}
	subnet, exists := rcvr.graph.Node(subnetReferences[0])
	return exists && subnet.Type == "aws_subnet" && rcvr.hasUniqueAttributeReference(subnet, "vpc_id", vpcAddress)
}

func (rcvr *awsRepository) nodeElement(node commonentity.Node) xaligoentity.XALElement {
	if group, ok := awsentity.LookupGroup(node.Type); ok && node.Kind == commonentity.NodeResource {
		return xaligoentity.XALElement{Tag: group.Tag, ID: rcvr.identities[node.Address], Title: nodeTitle(node)}
	}
	if mapping, ok := mappingFor(node); ok && node.Kind == commonentity.NodeResource {
		return xaligoentity.XALElement{Tag: "item", ID: mapping.CatalogID, Name: rcvr.identities[node.Address]}
	}
	if kind, ok := definitionKind(node.Kind); ok {
		if _, known := awsentity.LookupDefinition(kind, node.Type); known {
			return xaligoentity.XALElement{Tag: "rectangle", ID: rcvr.identities[node.Address], Title: node.Address}
		}
	}
	rcvr.diagnostics = append(rcvr.diagnostics, commonentity.Diagnostic{
		Code:     "MAPPING-W002",
		Severity: commonentity.SeverityWarning,
		Summary:  "Terraform block uses a generic XAL fallback",
		Detail:   fmt.Sprintf("No reviewed catalog mapping exists for %s; it is preserved as a rectangle.", node.Address),
		Range:    node.Range,
	})
	return xaligoentity.XALElement{Tag: "rectangle", ID: rcvr.identities[node.Address], Title: node.Address}
}

func definitionKind(kind commonentity.NodeKind) (awsentity.DefinitionKind, bool) {
	switch kind {
	case commonentity.NodeResource:
		return awsentity.DefinitionKindResource, true
	case commonentity.NodeData:
		return awsentity.DefinitionKindDataSource, true
	default:
		return "", false
	}
}

func mappingFor(node commonentity.Node) (awsentity.ItemMapping, bool) {
	loadBalancerType, _ := node.String("load_balancer_type")
	return awsentity.LookupItem(node.Type, loadBalancerType)
}

func isAWSNode(node commonentity.Node) bool {
	return strings.HasPrefix(node.Type, "aws_")
}

func awsRegionID(region string) string {
	if region == "" {
		return "xaligo-aws-region-unspecified"
	}
	return "xaligo-aws-region-" + NormalizeIdentifier(region)
}

func awsRegionFromAvailabilityZone(availabilityZone string) (string, bool) {
	availabilityZone = strings.ToLower(strings.TrimSpace(availabilityZone))
	if len(availabilityZone) < 2 {
		return "", false
	}
	last := availabilityZone[len(availabilityZone)-1]
	previous := availabilityZone[len(availabilityZone)-2]
	if last < 'a' || last > 'z' || previous < '0' || previous > '9' {
		return "", false
	}
	region := availabilityZone[:len(availabilityZone)-1]
	parts := strings.Split(region, "-")
	if len(parts) < 3 || len(parts) > 4 {
		return "", false
	}
	for _, part := range parts {
		if part == "" {
			return "", false
		}
	}
	return region, true
}

func (rcvr *awsRepository) hasUniqueAttributeReference(node commonentity.Node, attributeName, target string) bool {
	references := node.AttributeReferences(attributeName)
	matches := 0
	for _, reference := range references {
		if reference == target {
			matches++
		}
	}
	if matches != 1 {
		return false
	}
	if rcvr.usedReferences[node.Address] == nil {
		rcvr.usedReferences[node.Address] = map[string]bool{}
	}
	rcvr.usedReferences[node.Address][target] = true
	return true
}

func (rcvr *awsRepository) markReferenceUsed(source, target string) {
	if rcvr.usedReferences[source] == nil {
		rcvr.usedReferences[source] = map[string]bool{}
	}
	rcvr.usedReferences[source][target] = true
}

func (rcvr *awsRepository) reserveIdentity(identity, owner string) bool {
	if previous, exists := rcvr.identityOwners[identity]; exists {
		rcvr.diagnostics = append(rcvr.diagnostics, commonentity.Diagnostic{
			Code:     "MAPPING-E002",
			Severity: commonentity.SeverityError,
			Summary:  "Generated XAL identity collision",
			Detail:   fmt.Sprintf("Reserved id %q for %s conflicts with %s.", identity, owner, previous),
		})
		return false
	}
	rcvr.identityOwners[identity] = owner
	return true
}

func (rcvr *awsRepository) referencesAnySubnetInVPC(node commonentity.Node, vpcAddress string) bool {
	for _, reference := range node.AttributeReferences("subnets") {
		subnet, exists := rcvr.graph.Node(reference)
		if !exists || subnet.Type != "aws_subnet" {
			continue
		}
		if rcvr.hasUniqueAttributeReference(subnet, "vpc_id", vpcAddress) {
			if rcvr.usedReferences[node.Address] == nil {
				rcvr.usedReferences[node.Address] = map[string]bool{}
			}
			rcvr.usedReferences[node.Address][reference] = true
			return true
		}
	}
	return false
}

func (rcvr *awsRepository) reportOmittedDependencies() {
	for _, node := range rcvr.graph.Nodes {
		for _, dependency := range node.DependsOn {
			if _, exists := rcvr.graph.Node(dependency); !exists || rcvr.usedReferences[node.Address][dependency] {
				continue
			}
			rcvr.diagnostics = append(rcvr.diagnostics, commonentity.Diagnostic{
				Code:     "MAPPING-W003",
				Severity: commonentity.SeverityWarning,
				Summary:  "Terraform dependency is not rendered as traffic",
				Detail:   fmt.Sprintf("Dependency %s -> %s is retained in the graph but omitted from XAL because static references do not prove communication direction.", node.Address, dependency),
				Range:    node.Range,
			})
		}
	}
}

func nodeTitle(node commonentity.Node) string {
	name, known := node.ObjectString("tags", "Name")
	if !known || strings.TrimSpace(name) == "" {
		name = node.Address
	}
	if cidr, known := node.String("cidr_block"); known && strings.TrimSpace(cidr) != "" {
		return fmt.Sprintf("%s (%s)", name, cidr)
	}
	return name
}

// NormalizeIdentifier creates a stable XAL-safe identifier from a full
// Terraform address. Map rejects collisions rather than adding suffixes.
func NormalizeIdentifier(value string) string {
	var result strings.Builder
	dash := false
	for _, current := range strings.ToLower(strings.TrimSpace(value)) {
		switch {
		case unicode.IsLetter(current), unicode.IsDigit(current), current == '_':
			result.WriteRune(current)
			dash = false
		case current == '-', current == '.':
			if !dash && result.Len() > 0 {
				result.WriteByte('-')
				dash = true
			}
		default:
			if !dash && result.Len() > 0 {
				result.WriteByte('-')
				dash = true
			}
		}
	}
	return strings.Trim(result.String(), "-")
}
