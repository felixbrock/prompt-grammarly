package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/felixbrock/lemonai/internal/app"
	"github.com/felixbrock/lemonai/internal/component"
	"github.com/felixbrock/lemonai/internal/persistence"
)

/*
- Remove credentials from all files (Jack)
- Check for SQL injection
- Check for XSS
- Address prompt injection
- Block IP addresses: Too many requests
- change every 2s trigger to "until done" (Jack)
- Implement feedback logic
- Implement parent child tracing
- Implement custom instructions (Jack)
- Prioritize custom instructions (Jack)
- Connect to frontend (Jack)
- Remove prompts from code base (.gitignore file)
- Implement parent Id + Check where id needs to be passed (Jack)

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

	componentBuilder := app.ComponentBuilder{
		Index:   component.Index,
		App:     component.App,
		Draft:   component.DraftModeEditor,
		Edit:    component.EditModeEditor,
		Review:  component.ReviewModeEditor,
		Loading: component.Loading,
		Error:   component.Error,
	}

	dbHeader := []string{
		fmt.Sprintf("apikey: %s", config.DBApiKey),
		fmt.Sprintf("Authorization: Bearer %s", config.DBApiKey)}
	dbUrlBase := "https://cllevlrokigwvbbnbfiu.supabase.co/rest/v1"

	optRepo := persistence.OptimizationRepo{BaseHeaders: dbHeader, BaseUrl: fmt.Sprintf("%s/optimization", dbUrlBase)}
	suggRepo := persistence.SuggestionRepo{BaseHeaders: dbHeader, BaseUrl: fmt.Sprintf("%s/suggestion", dbUrlBase)}
	runRepo := persistence.RunRepo{BaseHeaders: dbHeader, BaseUrl: fmt.Sprintf("%s/run", dbUrlBase)}
	oaiRepo := persistence.OpenAIRepo{BaseHeaders: []string{
		"Content-Type:application/json",
		"OpenAI-Beta:assistants=v1",
		fmt.Sprintf("Authorization: Bearer %s", config.OAIApiKey)}}

	a := app.App{
		OptimimizationRepo: optRepo,
		RunRepo:            runRepo,
		SuggestionRepo:     suggRepo,
		OAIRepo:            oaiRepo,
		ComponentBuilder:   componentBuilder,
		Config:             config,
	}

	a.Start()
}
