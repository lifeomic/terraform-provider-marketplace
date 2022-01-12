# Example of how to use the resource

terraform {
  required_providers {
    marketplace = {
      version = "~> 1.0.0"
      source  = "lifeomic.com/tf/marketplace" # Doesn't mean anything
    }
  }
}

variable "name" {
  type = string
}

variable "app_tile_id" {
  type = string
}

variable "description" {
  type = string
}

provider "marketplace" {}

resource "app_tile" "anxiety" {
  provider       = marketplace
  name           = var.name
  description    = var.description
  image          = "icon-240.png"
  image_hash     = filemd5("./icon-240.png")
  app_tile_id    = var.app_tile_id
  version        = "0.0.11"
}
