package persistence

import (
	"encoding/json"

	domain "github.com/felixbrock/lemonai/internal/domain-xx"
)

type OptimizationRepo struct {
	ApiKey       string
	HeaderProtos []string
}

func (r OptimizationRepo) Insert(optimization domain.Optimization) (id string, err error) {
	body, err := json.Marshal(optimization)

	if err != nil {
		return "", err
	}

	resp, err := request[domain.Optimization](reqConfig{"POST", "https://cllevlrokigwvbbnbfiu.supabase.co/rest/v1/optimization", headerProtos, body}, 201)

	if err != nil {
		return "", err
	}

	return resp.Id, nil
}
