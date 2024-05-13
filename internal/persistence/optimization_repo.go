package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/felixbrock/prompt-grammarly/internal/app"
	"github.com/felixbrock/prompt-grammarly/internal/domain"
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

	_, err = request[domain.Optimization](context.TODO(), reqConfig{
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

	_, err = request[domain.Optimization](context.TODO(), reqConfig{
		Method:    "PATCH",
		Url:       r.BaseUrl,
		UrlParams: []string{fmt.Sprintf("id=eq.%s", id)},
		Body:      body,
		Headers:   append(r.BaseHeaders, "Content-Type:application/json")},
		204)

	if err != nil {
		return err
	}

	return nil
}

func (r OptimizationRepo) Read(id string) (*domain.Optimization, error) {
	records, err := request[[]domain.Optimization](context.TODO(), reqConfig{
		Method:    "GET",
		Url:       r.BaseUrl,
		UrlParams: []string{fmt.Sprintf("id=eq.%s", id)},
		Body:      nil,
		Headers:   r.BaseHeaders},
		200)

	if err != nil {
		return nil, err
	} else if len(*records) == 0 {
		return nil, errors.New("no optimization found")
	} else if len(*records) > 1 {
		return nil, errors.New("multiple optimizations found")
	}

	return &(*records)[0], nil
}
