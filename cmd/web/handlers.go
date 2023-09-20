package main

import (
	"fmt"
	"html/template"
	"net/http"
	"test-concurrency/data"
)

//admin@example.com
//verysecret

func (app *Config) HomePage(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "home.page.gohtml", nil)
}

func (app *Config) LoginPage(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "login.page.gohtml", nil)
}

func (app *Config) PostLoginPage(w http.ResponseWriter, r *http.Request) {
	_ = app.Session.RenewToken(r.Context()) //renovar o token da sessao
	
	//pegar dados do formulario
	err := r.ParseForm()
	if err != nil {
		app.ErrorLog.Println(err)
	}
	
	email := r.Form.Get("email")
	password := r.Form.Get("password")
	
	user, err := app.Models.User.GetByEmail(email)
	if err != nil {
		app.Session.Put(r.Context(), "error", "invalid credentials")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	
	//validar senha
	validPassword, err := user.PasswordMatches(password)
	if err != nil {
		app.Session.Put(r.Context(), "error", "invalid credentials")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	
	if !validPassword {
		//enviar email que houve uma tentativa de login
		msg := Message{
			To: email,
			Subject: "try to login with invalid password",
			Data: "If your dont tried to login, please verify your account",
		}
		app.sendEmail(msg)
		
		app.Session.Put(r.Context(), "error", "invalid credentials")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}	
	
	//logar usuario
	app.Session.Put(r.Context(), "userID", user.ID) //adicionar o id do usuario na sessao
	app.Session.Put(r.Context(), "user", user)

	//adicionar msg de login realizado com sucesso
	app.Session.Put(r.Context(), "flash", "Successful login")

	//redirecionar o usuario logado
	http.Redirect(w, r, "/", http.StatusSeeOther)

}

func (app *Config) Logout(w http.ResponseWriter, r *http.Request) {
	_ = app.Session.Destroy(r.Context())
	_=app.Session.RenewToken(r.Context())

	http.Redirect(w,r, "/login", http.StatusSeeOther)
}

func (app *Config) RegisterPage(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "register.page.gohtml", nil)

}

func (app *Config) PostRegisterPage(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.ErrorLog.Println(err)
	}

	u := data.User{
		Email: r.Form.Get("email"),
		FirstName: r.Form.Get("first-name"),
		LastName: r.Form.Get("last-name"),
		Password: r.Form.Get("password"),
		Active: 0,
		IsAdmin: 0,
	}

	_, err = u.Insert(u)
	if err != nil {
		app.Session.Put(r.Context(), "error", "Unable to create user")
		http.Redirect(w,r,"/register", http.StatusSeeOther)
		return
	}

	//enviar email de ativacao
	url := fmt.Sprintf("http://localhost/activate?email=%s", u.Email)
	signedUrl := GenerateTokenFromString(url)
	app.Infolog.Println(signedUrl)

	msg := Message{
		To: u.Email,
		Subject: "Activate your account",
		Template: "confirmation-email",
		Data: template.HTML(signedUrl),
	}

	app.sendEmail(msg)
	app.Session.Put(r.Context(), "flash", "email sending to confirm your signature")
	http.Redirect(w,r, "/login", http.StatusSeeOther)

}

func (app *Config) ActivateAccount(w http.ResponseWriter, r *http.Request) {
	url := r.RequestURI
	testUrl := fmt.Sprintf("http://localhost%s", url)
	okay := VerifyToken(testUrl)//verificar o token da url

	if !okay {
		app.Session.Put(r.Context(),"error", "invalid token")
		http.Redirect(w,r,"/", http.StatusSeeOther)
		return
	}
	
	u,err := app.Models.User.GetByEmail(r.URL.Query().Get("email"))
	if err != nil {
		app.Session.Put(r.Context(),"error", "user not found")
		http.Redirect(w,r,"/", http.StatusSeeOther)
		return
	}
	
	u.Active = 1
	err = u.Update()
	if err != nil {
		app.Session.Put(r.Context(),"error", "error to active user")
		http.Redirect(w,r,"/", http.StatusSeeOther)
		return
	}
	
	app.Session.Put(r.Context(),"flash", "account activated")
	http.Redirect(w,r,"/login", http.StatusSeeOther)

}
