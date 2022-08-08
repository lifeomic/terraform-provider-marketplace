package marketplace

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var testAccProviders = map[string]*schema.Provider{
	"marketplace": Provider(),
}