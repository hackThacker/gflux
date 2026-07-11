package main

import (
	"fmt"
	"os"
)

// Version is the current version of gflux, which can be injected at build time.
var Version = "dev"

var bannerPrinted bool

const bannerText = `
  ____ _____ _    _   ___  __
 / ___|  ___| |  | | | \ \/ /
| |  _| |_  | |  | | | |\  / 
| |_| |  _| | |__| |_| |/  \ 
 \____|_|   |_____\___//_/\_\

              gflux %s
              hackthacker.app

`

// printBanner prints the plain ASCII banner to os.Stderr.
func printBanner() {
	fmt.Fprintf(os.Stderr, bannerText, Version)
}

// showBanner prints the ASCII banner once unless silent is true.
func showBanner(silent bool) {
	if silent || bannerPrinted {
		return
	}
	printBanner()
	bannerPrinted = true
}
