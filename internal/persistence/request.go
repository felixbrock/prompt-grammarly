package persistence

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/felixbrock/lemonai/internal/app"
)

type reqConfig struct {
	Method    string
	Url       string
	UrlParams []string
	Headers   []string
	Body      []byte
}

func request[T any](ctx context.Context, config reqConfig, expectedResCode int) (*T, error) {
	url := config.Url
	if len(config.UrlParams) > 0 {
		url = fmt.Sprintf("%s?%s", config.Url, strings.Join(config.UrlParams, "&"))
	}
	req, err := http.NewRequestWithContext(ctx, config.Method, url, bytes.NewBuffer(config.Body))

	if err != nil {
		return nil, err
	}

	for i := 0; i < len(config.Headers); i++ {
		headerKV := strings.Split(config.Headers[i], ":")
		req.Header.Add(headerKV[0], headerKV[1])
	}

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, err
	} else if resp.StatusCode != expectedResCode {
		body, _ := app.Read(resp.Body)
		return nil, fmt.Errorf("unexpected response status code error: %s", body)
	}

	body, err := app.Read(resp.Body)

	if err != nil {
		return nil, err
	}

	if len(body) == 0 {
		return nil, nil
	}

	var t *T
	t, err = app.ReadJSON[T](body)

	if err != nil {
		return nil, err
	}

	return t, nil
}
