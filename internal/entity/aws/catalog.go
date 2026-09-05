package aws

type Scope uint8

const (
	ScopeUnspecified Scope = iota
	ScopeGlobal
	ScopeRegional
	ScopeVPC
	ScopeAvailabilityZone
)

type ItemMapping struct {
	CatalogID string
	Scope     Scope
}

type GroupMapping struct {
	Tag   string
	Scope Scope
}

// Catalog IDs are verified against xaligo's service-index.csv snapshot at
// commit 2333a3fff015ac3dd667037934b7eb5c4623cf8b.
var itemMappings = map[string]ItemMapping{
	"aws_apigatewayv2_api":      {CatalogID: "1176", Scope: ScopeRegional},
	"aws_cloudwatch_event_bus":  {CatalogID: "1495", Scope: ScopeRegional},
	"aws_cloudwatch_log_group":  {CatalogID: "1545", Scope: ScopeRegional},
	"aws_codebuild_project":     {CatalogID: "438", Scope: ScopeRegional},
	"aws_dynamodb_table":        {CatalogID: "1821", Scope: ScopeRegional},
	"aws_ecr_repository":        {CatalogID: "546", Scope: ScopeRegional},
	"aws_ecs_cluster":           {CatalogID: "547", Scope: ScopeRegional},
	"aws_glue_catalog_database": {CatalogID: "1433", Scope: ScopeRegional},
	"aws_iam_role":              {CatalogID: "1479", Scope: ScopeGlobal},
	"aws_instance":              {CatalogID: "1790", Scope: ScopeAvailabilityZone},
	"aws_internet_gateway":      {CatalogID: "1581", Scope: ScopeVPC},
	"aws_kms_alias":             {CatalogID: "217", Scope: ScopeRegional},
	"aws_kms_key":               {CatalogID: "217", Scope: ScopeRegional},
	"aws_lambda_function":       {CatalogID: "1783", Scope: ScopeRegional},
	"aws_lb":                    {CatalogID: "1182", Scope: ScopeVPC},
	"aws_lb_listener":           {CatalogID: "1182", Scope: ScopeVPC},
	"aws_nat_gateway":           {CatalogID: "1582", Scope: ScopeAvailabilityZone},
	"aws_db_instance":           {CatalogID: "117", Scope: ScopeAvailabilityZone},
	"aws_rds_cluster":           {CatalogID: "110", Scope: ScopeRegional},
	"aws_route53_zone":          {CatalogID: "1568", Scope: ScopeGlobal},
	"aws_s3_bucket":             {CatalogID: "1642", Scope: ScopeRegional},
	"aws_sqs_queue":             {CatalogID: "1508", Scope: ScopeRegional},
	"aws_ssm_parameter":         {CatalogID: "1529", Scope: ScopeRegional},
}

var groupMappings = map[string]GroupMapping{
	"aws_autoscaling_group":             {Tag: "auto-scaling-group", Scope: ScopeVPC},
	"aws_default_security_group":        {Tag: "security-group", Scope: ScopeVPC},
	"aws_elastic_beanstalk_environment": {Tag: "elastic-beanstalk-container", Scope: ScopeRegional},
	"aws_greengrass_group":              {Tag: "aws-iot-greengrass", Scope: ScopeRegional},
	"aws_greengrassv2_deployment":       {Tag: "aws-iot-greengrass-deployment", Scope: ScopeRegional},
	"aws_organizations_account":         {Tag: "aws-account", Scope: ScopeGlobal},
	"aws_security_group":                {Tag: "security-group", Scope: ScopeVPC},
	"aws_sfn_state_machine":             {Tag: "aws-step-functions-workflow", Scope: ScopeRegional},
	"aws_spot_fleet_request":            {Tag: "spot-fleet", Scope: ScopeRegional},
}

func LookupItem(resourceType, loadBalancerType string) (ItemMapping, bool) {
	mapping, ok := itemMappings[resourceType]
	if !ok {
		return ItemMapping{}, false
	}
	if resourceType != "aws_lb" {
		return mapping, true
	}
	switch loadBalancerType {
	case "application":
		mapping.CatalogID = "1592"
	case "gateway":
		mapping.CatalogID = "1594"
	case "network":
		mapping.CatalogID = "1595"
	}
	return mapping, true
}

func LookupGroup(resourceType string) (GroupMapping, bool) {
	mapping, ok := groupMappings[resourceType]
	return mapping, ok
}
