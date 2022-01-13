# <provider> Provider

This provider is for managing LifeOmic app store resources. Typically these applets are then published on the marketplace (marketplace provider is still under development). If you're working to use this provider for external app tiles, contact LifeOmic for assistance. A self-serve experience in under development.

The provider uses your local AWS config in order to authenticate. Support for token-based authentication with the public graphql-proxy will probably come in the future.

## Example Usage

```hcl
provider "marketplace" {}

resource "app_tile" "example" {
  provider       = marketplace
  name           = "Example App Tile for Marketplace"
  description    = "This applet is created and managed using terraform"
  author_display = "LifeOmic"
  app_tile_id    = "some_id" # Probably get this from a applet resource from appstore
  image          = "icon.png"
  image_hash     = filemd5("./icon.png")
  version        = "0.0.12"
}

resource "app_tile" "auto_version_example" {
  provider       = marketplace
  name           = "Example App Tile for Marketplace"
  description    = "This applet is created and managed using terraform"
  author_display = "LifeOmic"
  app_tile_id    = "some_id" # Probably get this from a applet resource from appstore
  image          = "icon.png"
  image_hash     = filemd5("./icon.png")
  auto_version   = true
}
```

## Argument Reference

* name: string
* description: string
* author_display: string
* app_tile_id: string
* image: string # Path to image
* image_hash: string # Hash so that we know when the image has changed
* version: string
* auto_version: bool # Will autoincrement the patch value on any change

