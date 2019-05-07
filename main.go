package main

import (
	"github.com/carfax/terraform-provider-infoblox/infoblox"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: infoblox.Provider})
}
