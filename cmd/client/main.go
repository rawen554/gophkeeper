package main

import (
	"log"

	"github.com/rawen554/goph-keeper/cmd/client/internal/cmd"
	"github.com/rawen554/goph-keeper/internal/logger"
)

func main() {
	logger, err := logger.NewLogger()
	if err != nil {
		log.Fatal(err)
	}

	if err := cmd.Execute(); err != nil {
		logger.Errorf("error: %v", err)
	}
}
