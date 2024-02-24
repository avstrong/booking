package main

import (
	"log"
	"os"

	"github.com/avstrong/booking/internal/app"
	"github.com/avstrong/booking/internal/logger"
)

func main() {
	l := logger.New(log.Default())

	var exitCode int

	if err := app.Run(l); err != nil {
		l.LogErrorf("Failed to run app: %v", err.Error())

		exitCode = 1
	}

	os.Exit(exitCode)
}
