package persistence

import (
	"encoding/json"
	"fmt"

	"github.com/felixbrock/lemonai/internal/app"
	"github.com/felixbrock/lemonai/internal/domain"
)

type RunRepo struct {
	BaseHeaders []string
	BaseUrl     string
}

func (r RunRepo) Insert(run domain.Run) error {
	body, err := json.Marshal(run)

	if err != nil {
		return err
	}

	_, err = request[domain.Run](reqConfig{
		Method:  "POST",
		Url:     r.BaseUrl,
		Body:    body,
		Headers: r.BaseHeaders},
		201)

	if err != nil {
		return err
	}

	return nil
}

func (r RunRepo) Update(id string, state string) error {
	body, err := json.Marshal(fmt.Sprintf(`{"state": "%s"}`, state))

	if err != nil {
		return err
	}

	_, err = request[domain.Run](reqConfig{
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

func (r RunRepo) Read(filter app.RunReadFilter) (*[]domain.Run, error) {
	records, err := request[[]domain.Run](reqConfig{
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
