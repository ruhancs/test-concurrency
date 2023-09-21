package main

import (
	"database/sql"
	"log"
	"sync"
	"test-concurrency/data"

	"github.com/alexedwards/scs/v2"
)

type Config struct {
	Session *scs.SessionManager
	DB *sql.DB
	Infolog *log.Logger
	ErrorLog *log.Logger
	Wait *sync.WaitGroup
	Models data.Models
	Mailer Mail
	ErrorChan chan error
	ErrorChanDone chan bool 
}