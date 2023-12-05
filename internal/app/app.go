package app

import (
	"log"
	"net/http"

	"github.com/a-h/templ"
)

func App() {

	http.Handle("/static/",
		http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	homeTempl := home()
	http.Handle("/", templ.Handler())

	http.Handle("/app", AppHandler(app))
	http.Handle("/chat", AppHandler(chat))

	log.Println("App running on 8000...")
	log.Fatal(http.ListenAndServe(":8000", nil))
}
