package persistence

import (
	"encoding/json"

	"github.com/felixbrock/lemonai/internal/domain"
)

type OptimizationRepo struct {
	BaseHeaders []string
}

func (r OptimizationRepo) Insert(optimization domain.Optimization) error {
	body, err := json.Marshal(optimization)

	if err != nil {
		return err
	}

	_, err = request[domain.Optimization](reqConfig{Method: "POST", Url: "https://cllevlrokigwvbbnbfiu.supabase.co/rest/v1/optimization", Body: body, Headers: r.BaseHeaders}, 201)

	if err != nil {
		return err
	}

	return nil
}
