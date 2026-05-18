// Copyright @ Vivicta. All Rights Reserved. 2026
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/Vivicta-SC/terraform-provider-ocp/internal/provider"
)

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "hashicorp.com/Vivicta-SC/ocp",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New("0.1.2"), opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
