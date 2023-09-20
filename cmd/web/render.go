package main

import (
	"fmt"
	"html/template"
	"net/http"
	"time"
)

var pathTemplates = "./cmd/web/templates"

type TemplateData struct {
	StingMap map[string]string
	IntMap map[string]int
	FloatMap map[string]float64
	Data map[string]any
	Flash string //msg que aparecerao somente uma vez
	Warning string
	Error string
	Authenticated bool
	Now time.Time
}

func (app *Config) render(w http.ResponseWriter, r *http.Request, t string, td *TemplateData) {
	partials := []string{
		fmt.Sprintf("%s/base.layout.gohtml", pathTemplates),
		fmt.Sprintf("%s/header.partial.gohtml", pathTemplates),
		fmt.Sprintf("%s/navbar.partial.gohtml", pathTemplates),
		fmt.Sprintf("%s/footer.partial.gohtml", pathTemplates),
		fmt.Sprintf("%s/alerts.partial.gohtml", pathTemplates),
	}

	var templateSlice []string
	templateSlice = append(templateSlice, fmt.Sprintf("%s/%s", pathTemplates,t))

	for _, x := range partials {
		templateSlice = append(templateSlice, x)
	}

	if td == nil {
		td = &TemplateData{}
	}

	tmpl,err := template.ParseFiles(templateSlice...)
	if err != nil {
		app.ErrorLog.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w,app.AddDefaultData(td,r)); err != nil {
		app.ErrorLog.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (app *Config) AddDefaultData(td *TemplateData, r *http.Request) *TemplateData {
	td.Flash = app.Session.PopString(r.Context(),"flash")//msg que aparecerao somente uma vez
	td.Warning = app.Session.PopString(r.Context(),"warning")
	td.Error = app.Session.PopString(r.Context(),"error")
	
	//verificar autenticacao
	if app.IsAuthenticated(r) {
		td.Authenticated = true
	}
	td.Now = time.Now()
	
	return td
}

func (app *Config) IsAuthenticated(r *http.Request)bool {
	return app.Session.Exists(r.Context(), "userID")
}