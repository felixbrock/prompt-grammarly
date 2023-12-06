package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

type component interface {
	Render(ctx context.Context, w io.Writer) error
}

type ComponentResponse struct {
	Error       error
	Message     string
	Code        int
	ContentType string
	Component   component
}

type ComponentHandler func(http.ResponseWriter, *http.Request) *ComponentResponse

func (ch ComponentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp := ch(w, r)

	if resp.Error != nil {
		slog.Error(fmt.Sprintf(`Error occured: %s`, resp.Error.Error()))
		http.Error(w, resp.Message, resp.Code)
	}

	if resp.Code != 0 {
		w.WriteHeader(resp.Code)
	}
	w.Header().Add("Content-Type", resp.ContentType)
	err := resp.Component.Render(r.Context(), w)

	if err != nil {
		slog.Error(fmt.Sprintf(`Error occured: %s`, err.Error()))
		http.Error(w, "templ: failed to render template", 500)
	}
}
