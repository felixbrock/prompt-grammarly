package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/felixbrock/lemonai/internal/app"
	"github.com/felixbrock/lemonai/internal/component"
	"github.com/felixbrock/lemonai/internal/persistence"
	_ "go.uber.org/automaxprocs"
)

/*
- Clean up Github
- Change all private keys due to github history
- Remove todo

LATER:
- Address prompt injection
- Implement feedback logic
- update env base and add assistant ids
- Implement feedback handling
*/

func devConfig() (*app.Config, error) {
	env, err := os.ReadFile("env.json")
	if err != nil {
		return nil, err
	}

	var config app.Config
	if err := json.Unmarshal(env, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func devHandler() {
	config, err := devConfig()

	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	baseHandler(config)
}

func prodConfig() (*app.Config, error) {
	config := app.Config{
		Env:       os.Getenv("ENV"),
		Port:      os.Getenv("PORT"),
		DBApiKey:  os.Getenv("DB_API_KEY"),
		DBUrl:     os.Getenv("DB_URL"),
		OAIApiKey: os.Getenv("OAI_API_KEY"),
		PHApiKey:  os.Getenv("PH_API_KEY"),
	}

	return &config, nil
}

func prodHandler() {
	config, err := prodConfig()

	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	baseHandler(config)
}

func baseHandler(config *app.Config) {

	componentBuilder := app.ComponentBuilder{
		Index:   component.Index,
		App:     component.App,
		Draft:   component.DraftModeEditor,
		Edit:    component.EditModeEditor,
		Loading: component.Loading,
		Error:   component.Error,
	}

	dbHeader := []string{
		fmt.Sprintf("apikey:%s", config.DBApiKey),
		fmt.Sprintf("Authorization:Bearer %s", config.DBApiKey)}
	dbUrlBase := "https://cllevlrokigwvbbnbfiu.supabase.co/rest/v1"

	optRepo := persistence.OptimizationRepo{BaseHeaders: dbHeader, BaseUrl: fmt.Sprintf("%s/optimization", dbUrlBase)}
	suggRepo := persistence.SuggestionRepo{BaseHeaders: dbHeader, BaseUrl: fmt.Sprintf("%s/suggestion", dbUrlBase)}
	runRepo := persistence.RunRepo{BaseHeaders: dbHeader, BaseUrl: fmt.Sprintf("%s/run", dbUrlBase)}

	oaiRepo := persistence.OAIRepo{BaseHeaders: []string{
		"Content-Type:application/json",
		"OpenAI-Beta:assistants=v1",
		fmt.Sprintf("Authorization: Bearer %s", config.OAIApiKey)}}
	phRepo := persistence.PHRepo{BaseHeaders: []string{"Content-Type: application/json"}, ApiKey: config.PHApiKey}

	repo := app.Repo{
		OpRepo:   optRepo,
		RunRepo:  runRepo,
		SuggRepo: suggRepo,
		OAIRepo:  oaiRepo,
		PHRepo:   phRepo,
	}

	a := app.App{
		Repo:             repo,
		ComponentBuilder: componentBuilder,
		Config:           *config,
	}

	a.Start()
}

func main() {
	env := os.Getenv("ENV")
	if env == "" {
		env = "dev"
	}

	switch env {
	case "dev":
		devHandler()
	case "prod":
		prodHandler()
	default:
		slog.Error("ENV not set")
		os.Exit(1)
	}
}
