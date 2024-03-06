package main

import (
	"context"
	"flag"
	"log"

	"terraform-provider-restapi/internal/provider"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

//go:generate go run github.com/opentofu/opentofu/cmd/tofu@latest fmt -recursive ./examples/
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest generate -provider-name restapi

var version = "dev"

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/hashicorp/restapi",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(version), opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
