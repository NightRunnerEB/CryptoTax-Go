package main

import (
	"log"

	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/app"
	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/config"
)

func main() {
	cfg, err := config.NewConfig("config.yaml")
	if err != nil {
		log.Fatalf("Config error: %s", err)
	}

	app.Run(cfg)
}
