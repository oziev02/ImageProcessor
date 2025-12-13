package main

import (
	"log"

	"github.com/oziev02/ImageProcessor/internal/app"
)

func main() {
	application, err := app.New()
	if err != nil {
		log.Fatalf("failed to create app: %v", err)
	}

	if err := application.Start(); err != nil {
		log.Fatalf("application error: %v", err)
	}
}
