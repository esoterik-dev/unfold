package main

import (
	"os"
	"os/signal"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

	"github.com/esoterik-dev/unfold/cmd"
)

// version is set via ldflags at build time: -ldflags "-X main.version=<tag>"
// Falls back to "dev" for local go install without ldflags.
var version = "dev"

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			log.Error().Msg("SIGINT recieved, shutting down")
			viper.WriteConfig()
			os.Exit(1)
		}
	}()

	defer viper.WriteConfig()

	cmd.Version = version
	cmd.Execute()
}
