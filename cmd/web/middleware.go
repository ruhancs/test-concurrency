package main

import "net/http"

func (app *Config) SessionLoad(next http.Handler) http.Handler {
	return app.Session.LoadAndSave(next)
}

func (app *Config) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//buscar o userID na sessao
		if !app.Session.Exists(r.Context(), "userID"){
			app.Session.Put(r.Context(), "warning", "You must log in to access this page")
			http.Redirect(w,r,"/login", http.StatusTemporaryRedirect)
			return
		} 
		next.ServeHTTP(w,r)
	})
}