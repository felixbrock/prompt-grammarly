package apphandler

import (
	"fmt"
	"log/slog"
	"net/http"
)

type AppError struct {
	Error   error
	Message string
	Code    int
}

type AppHandler func(http.ResponseWriter, *http.Request) *AppError

func (fn AppHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if e := fn(w, r); e != nil {
		slog.Error(fmt.Sprintf(`Error occured: %s`, e.Error.Error()))
		http.Error(w, e.Message, e.Code)
	}
}
