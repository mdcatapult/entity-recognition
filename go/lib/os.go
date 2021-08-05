package lib

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"
)

func HandleInterrupt() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	log.Fatal().Msg("process interrupted")
}
