package persistence

import (
	"encoding/json"
	"fmt"

	"github.com/felixbrock/lemonai/internal/app"
	"github.com/felixbrock/lemonai/internal/domain"
)

type SuggestionRepo struct {
	BaseHeaders []string
	BaseUrl     string
}

func (r SuggestionRepo) Insert(suggestions []domain.Suggestion) error {
	body, err := json.Marshal(suggestions)

	if err != nil {
		return err
	}

	_, err = request[domain.Suggestion](reqConfig{
		Method:  "POST",
		Url:     r.BaseUrl,
		Body:    body,
		Headers: r.BaseHeaders}, 201)

	if err != nil {
		return err
	}

	return nil
}

func (r SuggestionRepo) Update(id string, userFeedback string) error {
	body, err := json.Marshal(fmt.Sprintf(`{"user_feedback": "%s"}`, userFeedback))

	if err != nil {
		return err
	}

	_, err = request[domain.Suggestion](reqConfig{
		Method:    "PATCH",
		Url:       r.BaseUrl,
		UrlParams: []string{fmt.Sprintf("id=eq.%s", id)},
		Body:      body,
		Headers:   r.BaseHeaders},
		201)

	if err != nil {
		return err
	}

	return nil
}

func (r SuggestionRepo) Read(filter app.SuggestionReadFilter) (*[]domain.Suggestion, error) {
	records, err := request[[]domain.Suggestion](reqConfig{
		Method:    "GET",
		Url:       r.BaseUrl,
		UrlParams: []string{fmt.Sprintf("optimization_id=eq.%s", filter.OptimizationId)},
		Body:      nil,
		Headers:   r.BaseHeaders},
		200)

	if err != nil {
		return nil, err
	}

	return records, nil
}
