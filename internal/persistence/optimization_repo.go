package persistence

import (
	"encoding/json"
	"fmt"

	"github.com/felixbrock/lemonai/internal/app"
	"github.com/felixbrock/lemonai/internal/domain"
)

type OptimizationRepo struct {
	BaseHeaders []string
	BaseUrl     string
}

func (r OptimizationRepo) Insert(optimization domain.Optimization) error {
	body, err := json.Marshal(optimization)

	if err != nil {
		return err
	}

	_, err = request[domain.Optimization](reqConfig{
		Method:  "POST",
		Url:     r.BaseUrl,
		Body:    body,
		Headers: append(r.BaseHeaders, "Content-Type:application/json")},
		201)

	if err != nil {
		return err
	}

	return nil
}

func (r OptimizationRepo) Update(id string, opts app.OpUpdateOpts) error {
	body, err := json.Marshal(opts)

	if err != nil {
		return err
	}

	_, err = request[domain.Optimization](reqConfig{
		Method:    "PATCH",
		Url:       r.BaseUrl,
		UrlParams: []string{fmt.Sprintf("id=eq.%s", id)},
		Body:      body,
		Headers:   append(r.BaseHeaders, "Content-Type:application/json")},
		201)

	if err != nil {
		return err
	}

	return nil
}

func (r OptimizationRepo) Read(id string) (*domain.Optimization, error) {
	record, err := request[domain.Optimization](reqConfig{
		Method:    "GET",
		Url:       r.BaseUrl,
		UrlParams: []string{fmt.Sprintf("id=eq.%s", id)},
		Body:      nil,
		Headers:   r.BaseHeaders},
		200)

	if err != nil {
		return nil, err
	}

	return record, nil
}
