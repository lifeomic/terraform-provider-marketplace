package marketplace

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func getHash(url string) (*string, error) {
	res, err := http.Get(url)

	if err != nil {
		return nil, err
	}
	body := &bytes.Buffer{}
	_, err = body.ReadFrom(res.Body)

	hash := md5.Sum(body.Bytes())
	text := hex.EncodeToString(hash[:])
	return &text, nil
}

func readAppTile(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*MarketplaceClient)
	id := d.Id()
	app, err := client.getAppTileModule(id)
	if err != nil {
		return err
	}
	if app.Image != nil {
		hash, err := getHash(app.Image.Url)
		if err != nil {
			return err
		}
		d.Set("image_hash", hash)
		d.Set("image", app.Image.FileName+"."+app.Image.FileExtension)
	} else {
		d.Set("image_hash", nil)
		d.Set("image", nil)
	}

	d.Set("name", app.Name)
	d.Set("description", app.Description)
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
				Optional: true,
			},
			"image_hash": {
				Type:     schema.TypeString,
				Optional: true,
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
