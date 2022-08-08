package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"github.com/lifeomic/terraform-provider-marketplace/marketplace"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: marketplace.Provider,
	})
}
