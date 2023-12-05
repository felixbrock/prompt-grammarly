package app

import (
	"log"
	"net/http"

	apphandler "github.com/felixbrock/lemonai/internal/appHandler"
)

func App() {

	http.Handle("/dist/static/",
		http.StripPrefix("/dist/static/", http.FileServer(http.Dir("dist/static"))))
	http.Handle("/", apphandler.AppHandler(home))
	http.Handle("/app", apphandler.AppHandler(app))
	http.Handle("/chat", apphandler.AppHandler(chat))

	log.Println("App running on 8000...")
	log.Fatal(http.ListenAndServe(":8000", nil))
}
