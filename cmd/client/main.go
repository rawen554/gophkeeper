package main

import (
	"fmt"

	"github.com/rawen554/goph-keeper/cmd/client/internal/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Printf("error: %v", err)
	}
}
