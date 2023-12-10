package app

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

type appConfig struct {
	Port string
}

func config() appConfig {
	port := os.Getenv("GOPORT")
	if port == "" {
		port = "8000"
	}

	return appConfig{Port: port}
}

func App() {

	http.Handle("/static/",
		http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	// the
	http.Handle("/", ComponentHandler(index))
	http.Handle("/app", ComponentHandler(app))
	http.Handle("/editor/draft", ComponentHandler(draftModeEditor))
	http.Handle("/editor/edit", ComponentHandler(editModeEditor))
	http.Handle("/editor/review", ComponentHandler(reviewModeEditor))
	http.Handle("/optimize", ComponentHandler(optimize))
	http.Handle("/chat", ComponentHandler(chat))

	config := config()
	log.Printf("App running on %s...", config.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", config.Port), nil))
}
