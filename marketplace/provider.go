package marketplace

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	return BuildAppStoreClient()
}

func Provider() *schema.Provider {
	return &schema.Provider{
		ConfigureFunc: providerConfigure,
		Schema:        map[string]*schema.Schema{},
		ResourcesMap: map[string]*schema.Resource{
			"app_tile":          appTileResource(),
			"wellness_offering": wellnessOfferingResource(),
		},
	}
}
