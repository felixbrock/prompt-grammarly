package persistence

import (
	"context"
	"fmt"
)

type PHRepo struct {
	BaseHeaders []string
	ApiKey      string
}

func (r PHRepo) Capture(eventType string, opid string) error {
	url := "https://eu.posthog.com/capture/"
	body := []byte(fmt.Sprintf(`{
		"api_key": "%s",
		"event": "%s",
		"properties": {
			"distinct_id": "%s"}}`, r.ApiKey, eventType, opid))

	_, err := request[struct{}](context.TODO(), reqConfig{Method: "POST", Url: url, Headers: r.BaseHeaders, Body: body}, 200)

	if err != nil {
		return err
	}

	return nil
}
