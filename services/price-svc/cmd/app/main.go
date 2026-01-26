package main

import (
	"log"

	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/app"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/config"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Config error: %s", err)
	}

	app.Run(cfg)
}
