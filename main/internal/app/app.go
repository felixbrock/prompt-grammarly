package app

import (
	"log"
	"net/http"

	apphandler "github.com/felixbrock/lemonai/internal/appHandler"
)

func App() {
	// http.HandleFunc("/", home)

	http.Handle("/eval", apphandler.AppHandler(eval))

	// http.HandleFunc("/clicked", clicked)

	// http.HandleFunc("/team", team)

	log.Println("App running on 8000...")
	log.Fatal(http.ListenAndServe(":8000", nil))
}
