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

variable "image" {
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
  image          = var.image
  app_tile_id    = var.app_tile_id
  version        = "0.0.1"
}
