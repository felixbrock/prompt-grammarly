package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/felixbrock/lemonai/internal/app"
	"github.com/felixbrock/lemonai/internal/persistence"
)

/*
- Remove credentials from all files (Jack)
- Check for SQL injection
- Check for XSS
- Block IP addresses: Too many requests
- change every 2s trigger to "until done" (Jack)
- Implement feedback logic
- Implement parent child tracing
- Implement custom instructions (Jack)
- Prioritize custom instructions (Jack)
- Connect to frontend (Jack)

- Host (Jack)
- Make sure db is secure (Jack)
- Add Posthog
*/

func config() app.Config {
	port := os.Getenv("GOPORT")
	if port == "" {
		port = "8000"
	}

	oaiApiKey := os.Getenv("OAI_API_KEY")
	if oaiApiKey == "" {
		slog.Error("OAI_API_KEY environment variable not set")
	}

	dbApiKey := os.Getenv("DB_API_KEY")
	if dbApiKey == "" {
		slog.Error("DB_API_KEY environment variable not set")
	}

	return app.Config{Port: port, OAIApiKey: oaiApiKey, DBApiKey: dbApiKey}
}

func main() {
	config := config()

	optRepo := persistence.OptimizationRepo{BaseHeaders: []string{
		fmt.Sprintf("apikey: %s", config.DBApiKey),
		fmt.Sprintf("Authorization: Bearer %s", config.DBApiKey)}}
	oaiRepo := persistence.OpenAIRepo{BaseHeaders: []string{
		"Content-Type:application/json",
		"OpenAI-Beta:assistants=v1",
		fmt.Sprintf("Authorization: Bearer %s", config.OAIApiKey)}}

	a := app.App{
		OptimimizationRepo: optRepo,
		OAIRepo:            oaiRepo,

		Config: config,
	}

	a.Start()
}
