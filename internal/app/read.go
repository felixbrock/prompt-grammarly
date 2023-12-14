package app

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
)

func Read(reader io.ReadCloser) ([]byte, error) {
	var err error

	defer func() {
		err = (reader).Close()
		if err != nil {
			slog.Error(fmt.Sprintf("Error occured: %s", err.Error()))
		}
	}()

	var content []byte
	content, err = io.ReadAll(reader)

	if err != nil {
		return nil, err
	}

	return content, nil
}

func ReadJSON[T any](content []byte) (*T, error) {
	var t *T
	err := json.Unmarshal(content, &t)

	if err != nil {
		return nil, err
	}

	return t, nil
}
