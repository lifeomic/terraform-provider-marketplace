package marketplace

import (
	"errors"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func readAppTile(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*MarketplaceClient)
	id := d.Id()
	app, err := client.getAppTileModule(id)
	if err != nil {
		return err
	}
	d.Set("name", app.Name)
	d.Set("description", app.Description)
	d.Set("image", "something_bogus")
	d.Set("app_tile_id", app.Source.Id)
	return nil
}

func createAppTile(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*MarketplaceClient)
	id, err := client.publishNewAppTileModule(appTileCreate{
		Name:           d.Get("name").(string),
		Image:          d.Get("image").(string),
		AppTileId:      d.Get("app_tile_id").(string),
		Description:    d.Get("description").(string),
		Version:        d.Get("version").(string),
		ParentModuleId: nil,
	})
	println("err", err)
	if err != nil {
		return err
	}
	println("setting id", *id)
	d.SetId(*id)
	return nil
}

func updateAppTile(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*MarketplaceClient)
	id := d.Id()
	_, err := client.publishNewAppTileModule(appTileCreate{
		Name:           d.Get("name").(string),
		Image:          d.Get("image").(string),
		AppTileId:      d.Get("app_tile_id").(string),
		Description:    d.Get("description").(string),
		Version:        d.Get("version").(string),
		ParentModuleId: &id,
	})
	return err
}

func deleteAppTile(d *schema.ResourceData, meta interface{}) error {
	return errors.New("Unimplemented")
}

func appTileResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Required: true,
			},
			"image": {
				Type:     schema.TypeString,
				Required: true,
			},
			"app_tile_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"version": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
		Create: createAppTile,
		Read:   readAppTile,
		Update: updateAppTile,
		Delete: deleteAppTile,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}
