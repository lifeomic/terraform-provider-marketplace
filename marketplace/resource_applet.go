package marketplace

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/coreos/go-semver/semver"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func getHash(url string) (*string, error) {
	res, err := http.Get(url)

	if err != nil {
		return nil, err
	}
	body := &bytes.Buffer{}
	_, err = body.ReadFrom(res.Body)
	if err != nil {
		return nil, err
	}

	hash := md5.Sum(body.Bytes())
	text := hex.EncodeToString(hash[:])
	return &text, nil
}

func getCustomClient(d *schema.ResourceData, client *MarketplaceClient) (*MarketplaceClient, error) {
	account, ok := d.Get("account").(string)
	if !ok || account == "" {
		return client, nil
	}

	if d.HasChange("account") {
		old, _ := d.GetChange("account")
		oldAccount, ok := old.(string)
		if ok && oldAccount != "" {
			account = oldAccount
		}
	}

	c, err := buildCustomClient(account, defaultUser, defaultPolicy)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func inputFromState(d *schema.ResourceData, id *string) appTileCreate {
	account, ok := d.Get("account").(*string)
	if !ok {
		account = nil
	}

	scope, ok := d.Get("scope").(string)
	if !ok {
		scope = ""
	}

	parentModuleId, ok := d.Get("parent_module_id").(*string)
	if !ok {
		parentModuleId = nil
	}
	// allow overriding the id which is necessary for updating a module
	if id != nil {
		parentModuleId = id
	}

	url, ok := d.Get("url").(string)
	if !ok {
		url = ""
	}
	return appTileCreate{
		Name:           d.Get("name").(string),
		Image:          d.Get("image").(string),
		AppTileId:      d.Get("app_tile_id").(string),
		Description:    d.Get("description").(string),
		Version:        d.Get("version").(string),
		Account:        account,
		Scope:          &scope,
		ParentModuleId: parentModuleId,
		Url:            &url,
	}

}

func readAppTile(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*MarketplaceClient)
	id := d.Id()
	var app *AppTileModule
	retryCount := 2

	client, err := getCustomClient(d, client)
	if err != nil {
		return err
	}

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
		return errors.New("no Module Found")
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
	d.Set("version", app.Version)
	if source, ok := app.Source.(*AppTileModuleSourceAppTile); ok {
		d.Set("app_tile_id", source.Id)
	}
	d.Set("account", app.Organization.Id)
	d.Set("scope", app.Scope)
	return nil
}

func createAppTile(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*MarketplaceClient)
	_, versionExists := d.GetOk("version")
	if !versionExists {
		if !d.Get("auto_version").(bool) {
			return errors.New("if you don't specify a version, you must use auto_version")
		}
		d.Set("version", "0.0.0")
	}
	client, err := getCustomClient(d, client)
	if err != nil {
		return err
	}

	id, err := client.publishNewAppTileModule(inputFromState(d, nil))
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

	client, err := getCustomClient(d, client)
	if err != nil {
		return err
	}

	_, err = client.publishNewAppTileModule(inputFromState(d, &id))
	return err
}

func deleteAppTile(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*MarketplaceClient)
	id := d.Id()

	client, err := getCustomClient(d, client)
	if err != nil {
		return err
	}

	if _, err := DeleteModule(context.Background(), client.gqlClient, DeleteModuleInput{ModuleId: id}); err != nil {
		return fmt.Errorf("failed to delete module %s: %w", id, err)
	}

	d.SetId("")
	return nil
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
			"account": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"scope": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "One of 'LICENSED, 'ORGANIZATION', or 'PUBLIC'. Defaults to 'PUBLIC'",
			},
			"url": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Required when scope is set to 'ORGANIZATION'",
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
