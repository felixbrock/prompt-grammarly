package persistence

import (
	"bytes"
	"errors"
	"net/http"
	"strings"

	"github.com/felixbrock/lemonai/internal/app"
)

type reqConfig struct {
	Method  string
	Url     string
	Headers []string
	Body    []byte
}

func request[T any](config reqConfig, expectedResCode int) (*T, error) {
	req, err := http.NewRequest(config.Method, config.Url, bytes.NewBuffer(config.Body))

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
		return nil, errors.New("unexpected response status code error")
	}

	body, err := app.Read(resp.Body)

	if err != nil {
		return nil, err
	}

	var t *T
	t, err = app.ReadJSON[T](body)

	if err != nil {
		return nil, err
	}

	return t, nil
}
