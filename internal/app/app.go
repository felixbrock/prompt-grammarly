package app

import (
	"log"
	"net/http"
)

func App() {

	http.Handle("/static/",
		http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	// the
	http.Handle("/", ComponentHandler(index))
	http.Handle("/app", ComponentHandler(app))
	http.Handle("/editor/draft", ComponentHandler(draftModeEditor))
	http.Handle("/editor/edit", ComponentHandler(editModeEditor))
	http.Handle("/editor/review", ComponentHandler(reviewModeEditor))
	http.Handle("/chat", ComponentHandler(chat))

	log.Println("App running on 8000...")
	log.Fatal(http.ListenAndServe(":8000", nil))
}
