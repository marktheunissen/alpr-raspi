package config

import (
	"log"
	"os"

	flags "github.com/jessevdk/go-flags"
)

// Options describes all the CLI flags that can be passed
type Options struct {
	WatchDir string `env:"WATCHER_DIR" required:"true" default:"./testdata" short:"a"`
}

// Opts is the application config struct that we allow external access too
var Opts Options

func init() {
	_, err := flags.Parse(&Opts)
	if err != nil {
		log.Println(err)
		log.Println("Missing ENV vars containing configuration, try `. lpr.env`")
		os.Exit(1)
	}
}
