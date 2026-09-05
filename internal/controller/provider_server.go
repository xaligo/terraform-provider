package controller

import (
	"context"
	"fmt"

	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

const ProviderAddress = "registry.terraform.io/xaligo/xaligo"

type ProviderServerController interface {
	Serve(context.Context, func() frameworkprovider.Provider, bool) error
	ParseDebug([]string) (bool, error)
}

type providerServerController struct{}

func NewProviderServerController() ProviderServerController {
	return &providerServerController{}
}

func (rcvr *providerServerController) Serve(ctx context.Context, factory func() frameworkprovider.Provider, debug bool) error {
	return providerserver.Serve(ctx, factory, providerserver.ServeOpts{
		Address:         ProviderAddress,
		Debug:           debug,
		ProtocolVersion: 6,
	})
}

func (rcvr *providerServerController) ParseDebug(arguments []string) (bool, error) {
	debug := false
	for _, argument := range arguments {
		switch argument {
		case "-debug", "--debug":
			debug = true
		default:
			return false, fmt.Errorf("unknown provider server argument %q", argument)
		}
	}
	return debug, nil
}
