package marketplace

import (
	"errors"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var errNotImplemented = errors.New("resource handler not implemented yet")

func createWellnessOffering(d *schema.ResourceData, meta interface{}) error {
	return errNotImplemented
}

func readWellnessOffering(d *schema.ResourceData, meta interface{}) error {
	return errNotImplemented

}

func updateWellnessOffering(d *schema.ResourceData, meta interface{}) error {
	return errNotImplemented
}

func deleteWellnessOffering(d *schema.ResourceData, meta interface{}) error {
	return errNotImplemented
}

func wellnessOfferingResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"title": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Required: true,
			},
			"marketplace_provider": {
				Type:     schema.TypeString,
				Required: true,
			},
			"version": {
				Type:     schema.TypeString,
				Required: true,
			},
			"auto_version": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"image_url": {
				Type:     schema.TypeString,
				Required: true,
			},
			"info_url": {
				Type:     schema.TypeString,
				Required: true,
			},
			"approximate_unit_cost_pennies": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"install_url": {
				Required: true,
			},
			"configuration_schema": {
				Type:     schema.TypeString,
				Required: true,
			},
			"is_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
		},
		Create: createWellnessOffering,
		Read:   readWellnessOffering,
		Update: updateWellnessOffering,
		Delete: deleteWellnessOffering,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}
