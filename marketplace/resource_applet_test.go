package marketplace

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const testAppletResName = "applet.test"

func TestAccResourceApplet_basic(t *testing.T) {
	t.Parallel()

	name := acctest.RandomWithPrefix("tf-test")
	// client := BuildAppStoreClient()

	resource.Test(t, resource.TestCase{
		IDRefreshName: testAppletResName,
		Providers:     testAccProviders,
		ExternalProviders: map[string]resource.ExternalProvider{
			"appstore": {Source: "lifeomic/appstore"},
		},

		Steps: []resource.TestStep{{
			Config: testAccResourceApplet_basic(name),
			Check: resource.ComposeAggregateTestCheckFunc(
				testAccCheckAppletExists,
				resource.TestCheckResourceAttr(testAppletResName, "name", name),
				resource.TestCheckResourceAttr(testAppletResName, "description", "this is a test"),
				resource.TestCheckResourceAttr(testAppletResName, "version", "0.0.0"),
				resource.TestCheckResourceAttr(testAppletResName, "auto_version", "true"),
				resource.TestCheckResourceAttrSet(testAppletResName, "app_tile_id"),
				resource.TestCheckResourceAttrSet(testAppletResName, "image"),
				resource.TestCheckResourceAttrSet(testAppletResName, "image_hash"),
			),
		}, {
			Config:             testAccResourceApplet_basic(name),
			PlanOnly:           true,
			ExpectNonEmptyPlan: false,
		}},
	})
}

func testAccCheckAppletExists(s *terraform.State) error {
	client, err := BuildAppStoreClient()
	if err != nil {
		return err
	}

	// Find the applet resource in the Terraform state.
	for _, res := range s.RootModule().Resources {
		if res.Type != "applet" {
			continue
		}

		// Ensure we can get the applet associated with this resource.
		if _, err := GetPublishedModule(context.Background(), client.gqlClient, res.Primary.ID, ""); err != nil {
			return err
		}

		return nil
	}

	// If we got to this point, the applet resource was not in the state
	// for some reason.
	return errors.New("could not find applet in state graph")
}

func testAccResourceApplet_basic(name string) string {
	imagePath, _ := filepath.Abs(filepath.Join("testdata", "small-image.png"))

	// Defining a provider block here and referencing it in the resource
	// declaration is necessary because this provider does not follow the
	// Terraform resource naming convention (so the Terraform cannot infer
	// the provider to use given the resource name).
	//
	// TODO: remove this hack once we properly name our resources
	return fmt.Sprintf(`
resource "app_tile" "test" {
	provider     = marketplace
	app_tile_id  = applet.test.id
	description  = applet.test.description
	name         = applet.test.name
	image        = "%s"
	image_hash   = filemd5("%[2]s")
	auto_version = true

	lifecycle {
		ignore_changes = [version]
	}
}`, name, imagePath)
}
