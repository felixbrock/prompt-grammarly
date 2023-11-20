package router

import (
	"fmt"
	"net/http"

	apphandler "github.com/felixbrock/lemonai/internal/appHandler"
)

func eval(w http.ResponseWriter, r *http.Request) *apphandler.AppError {
	v, err := http.Get("http://0.0.0.0:80")

	fmt.Println(v)
	fmt.Println(err)

	if err != nil {
		return &apphandler.AppError{Error: err, Message: "test went wrong", Code: 500}
	}
	return nil
}

// func home(w http.ResponseWriter, r *http.Request) {
// 	tmpl := template.Must(template.ParseFiles("./templates/index.html"))
// 	tmpl.Execute(w, nil)
// }

// func clicked(w http.ResponseWriter, r *http.Request) {
// 	tmpl := template.Must(template.ParseFiles("./templates/fragments/button.html"))
// 	tmpl.Execute(w, nil)
// }

// func team(w http.ResponseWriter, r *http.Request) {
// 	tmpl := template.Must(template.ParseFiles("./templates/fragments/button.html"))
// 	tmpl.Execute(w, nil)
// }
