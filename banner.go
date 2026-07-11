package main

import (
	"fmt"
	"os"
)

// Version is the current version of gflux, which can be injected at build time.
var Version = "dev"

var bannerPrinted bool

const bannerText = `
 ██████  ███████ ██      ██    ██ ██   ██ 
██       ██      ██      ██    ██  ██ ██  
██   ███ █████   ██      ██    ██   ███   
██    ██ ██      ██      ██    ██  ██ ██  
 ██████  ██      ███████  ██████  ██   ██ 
                                          
                                          

              gflux %s
              hackthacker.app

`

// showBanner prints the ASCII banner once unless silent is true.
func showBanner(silent bool) {
	if silent || bannerPrinted {
		return
	}
	fmt.Fprintf(os.Stderr, bannerText, Version)
	bannerPrinted = true
}
