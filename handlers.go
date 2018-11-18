package aggregator

import (
	"html/template"
	"log"
	"net/http"
)

var TMPL = template.Must(template.ParseFiles("templates/template.gohtml"))

func (agr Aggregator) HandleSearch(w http.ResponseWriter, r *http.Request) {

	if err := r.ParseForm(); err != nil {
		log.Printf("Error when try parse post form data: %s", err)
		TMPL.Execute(w, nil)
	}

	searchQuery, ok := r.Form["search"]
	if !ok {
		TMPL.Execute(w, nil)
	}

	result, err := agr.SearchRecord(searchQuery[0])
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Print(err)
	}

	TMPL.Execute(w, result)
}

func (agr Aggregator) IndexHandler(w http.ResponseWriter, r *http.Request) {
	TMPL.Execute(w, nil)
}
