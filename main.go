package main

import (
	"context"
	"flag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"log"
	"terraform-provider-localfile/internal"
)

// main starts the Terraform provider server.  A --debug flag allows
// running the provider in debug mode for use with IDE debuggers.
func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "set to true to enable debugging via Terraform provider framework")
	flag.Parse()
	err := providerserver.Serve(context.Background(), func() provider.Provider {
		return internal.NewProvider("dev")
	}, providerserver.ServeOpts{
		Address: "registry.terraform.io/example/localfile",
		Debug:   debug,
	})
	if err != nil {
		log.Fatal(err)
	}
}
