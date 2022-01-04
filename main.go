package main

import (
	"github.com/lifeomic/terraform-provider-marketplace/marketplace"

	"github.com/hashicorp/terraform-plugin-sdk/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{ProviderFunc: marketplace.Provider})
}
