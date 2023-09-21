package main

import (
	"context"
	"encoding/gob"
	"log"
	"net/http"
	"os"
	"sync"
	"test-concurrency/data"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
)

var testApp Config

func TestMain(m *testing.M) {
	gob.Register(data.User{})

	session := scs.New()
	session.Lifetime = 24 * time.Hour 
	session.Cookie.Persist = true
	session.Cookie.SameSite = http.SameSiteLaxMode
	session.Cookie.Secure = true

	testApp = Config{
		Session: session,
		DB: nil,
		Infolog: log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime),
		ErrorLog: log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile),
		Wait: &sync.WaitGroup{},
		ErrorChan: make(chan error),
		ErrorChanDone: make(chan bool),
	}

	errorChan := make(chan error)
	mailerChan := make(chan Message, 100)
	mailerDoneChan := make(chan bool)

	testApp.Mailer = Mail{
		Wait: testApp.Wait,
		ErrorChan: errorChan,
		MailerChan: mailerChan,
		DoneChan: mailerDoneChan,
	}

	go func() {
		select{
		case <- testApp.Mailer.MailerChan:
		case <-testApp.Mailer.ErrorChan:
		case <-testApp.Mailer.DoneChan:
			return
		}
	}()

	go func ()  {
		for {
			select{
			case err := <-testApp.ErrorChan:
				testApp.ErrorLog.Println(err)
			case <- testApp.ErrorChanDone:
				return
			}
		}
	}()

	os.Exit(m.Run())
}

func getCtx(req *http.Request) context.Context {
	ctx,err := testApp.Session.Load(req.Context(),req.Header.Get("X-Session"))
	if err != nil {
		log.Println(err)
	}
	return ctx
}