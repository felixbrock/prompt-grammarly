package app

import (
	"log"
	"net/http"

	apphandler "github.com/felixbrock/lemonai/internal/appHandler"
)

func App() {

	http.Handle("/", apphandler.AppHandler(chat))

	log.Println("App running on 8000...")
	log.Fatal(http.ListenAndServe(":8000", nil))
}
