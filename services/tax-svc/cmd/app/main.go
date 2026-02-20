package main

import (
	"log"

	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/app"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/config"
)

func main() {
	cfg, err := config.NewConfig("config.yaml")
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	app.Run(cfg)
}
