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

type AppResp struct {
	Error       error
	Message     string
	Code        int
	ContentType string
	Component   component
}

type Controller interface {
	Handle(http.ResponseWriter, *http.Request) *AppResp
}

type AppHandler struct {
	c Controller
}

func (h AppHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp := h.c.Handle(w, r)

	if resp.Error != nil {
		slog.Error(fmt.Sprintf(`Error occured: %s`, resp.Error.Error()))
	}

	if resp.Code != 0 {

		// Overwrite error code to allow for component rendering on client
		if resp.Code != 200 && resp.Code != 201 {
			resp.Code = 200
		}

		w.WriteHeader(resp.Code)
	}
	w.Header().Add("Content-Type", resp.ContentType)
	err := resp.Component.Render(r.Context(), w)

	if err != nil {
		slog.Error(fmt.Sprintf(`Error occured: %s`, err.Error()))
		http.Error(w, "templ: failed to render template", 500)
	}
}
