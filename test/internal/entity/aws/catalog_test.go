package aws_test

import (
	"testing"

	awsentity "github.com/xaligo/terraform-provider/internal/entity/aws"
)

func TestDedicatedAWSGroupMappingsUseMatchingXALTags(t *testing.T) {
	t.Parallel()

	tests := map[string]awsentity.GroupMapping{
		"aws_autoscaling_group":             {Tag: "auto-scaling-group", Scope: awsentity.ScopeVPC},
		"aws_default_security_group":        {Tag: "security-group", Scope: awsentity.ScopeVPC},
		"aws_elastic_beanstalk_environment": {Tag: "elastic-beanstalk-container", Scope: awsentity.ScopeRegional},
		"aws_greengrass_group":              {Tag: "aws-iot-greengrass", Scope: awsentity.ScopeRegional},
		"aws_greengrassv2_deployment":       {Tag: "aws-iot-greengrass-deployment", Scope: awsentity.ScopeRegional},
		"aws_organizations_account":         {Tag: "aws-account", Scope: awsentity.ScopeGlobal},
		"aws_security_group":                {Tag: "security-group", Scope: awsentity.ScopeVPC},
		"aws_sfn_state_machine":             {Tag: "aws-step-functions-workflow", Scope: awsentity.ScopeRegional},
		"aws_spot_fleet_request":            {Tag: "spot-fleet", Scope: awsentity.ScopeRegional},
	}
	for resourceType, want := range tests {
		resourceType, want := resourceType, want
		t.Run(resourceType, func(t *testing.T) {
			t.Parallel()
			got, ok := awsentity.LookupGroup(resourceType)
			if !ok || got != want {
				t.Fatalf("LookupGroup(%q) = %#v, %v; want %#v", resourceType, got, ok, want)
			}
		})
	}
	if _, ok := awsentity.LookupGroup("aws_s3_bucket"); ok {
		t.Fatal("LookupGroup found an item-mapped resource")
	}
}
