package marketplace

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/coreos/go-semver/semver"
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
	var app *appTileModule
	retryCount := 2
	for app == nil && retryCount > 0 {
		inner, err := client.getAppTileModule(id)
		app = inner
		if err != nil {
			return err
		}
		if app == nil {
			// Sometimes with eventual consistency the module isn't created yet
			log.Println("Module not found, trying again in 5 seconds...")
			time.Sleep(5 * time.Second)
		}
		retryCount -= 1
	}

	if app == nil {
		return errors.New("No Module Found")
	}

	if app.IconV2 != nil {
		hash, err := getHash(app.IconV2.Url)
		if err != nil {
			return err
		}
		d.Set("image_hash", hash)
	} else {
		d.Set("image_hash", nil)
	}

	d.Set("name", app.Title)
	d.Set("description", app.Description)
	d.Set("app_tile_id", app.Source.Id)
	d.Set("version", app.Version)
	return nil
}

func createAppTile(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*MarketplaceClient)
	_, versionExists := d.GetOk("version")
	if !versionExists {
		if !d.Get("auto_version").(bool) {
			return errors.New("If you don't specify a version, you must use auto_version")
		}
		d.Set("version", "0.0.0")
	}
	id, err := client.publishNewAppTileModule(appTileCreate{
		Name:           d.Get("name").(string),
		Image:          d.Get("image").(string),
		AppTileId:      d.Get("app_tile_id").(string),
		Description:    d.Get("description").(string),
		Version:        d.Get("version").(string),
		ParentModuleId: nil,
	})
	if err != nil {
		return err
	}
	d.SetId(*id)
	return readAppTile(d, meta)
}

func updateAppTile(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*MarketplaceClient)
	id := d.Id()
	if d.Get("auto_version").(bool) {
		version, err := semver.NewVersion(d.Get("version").(string))
		if err != nil {
			return err
		}
		version.BumpPatch()
		d.Set("version", version.String())
	}

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
				Optional: true,
			},
			"auto_version": {
				Type:     schema.TypeBool,
				Optional: true,
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
