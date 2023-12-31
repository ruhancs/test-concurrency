package main

import (
	"database/sql"
	"encoding/gob"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"test-concurrency/data"
	"time"

	"github.com/alexedwards/scs/redisstore"
	"github.com/alexedwards/scs/v2"
	"github.com/gomodule/redigo/redis"
	_ "github.com/jackc/pgconn"
	_ "github.com/jackc/pgx/v4"
	_ "github.com/jackc/pgx/v4/stdlib"
)

var webPort = "8000"

//password admin = verysecret

func main() {
	db := initDB()

	//criar sessao, inserir no redis
	session := initSession()

	//criar logs
	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	wg := sync.WaitGroup{}

	app := Config{
		Session: session,
		DB: db,
		Wait: &wg,
		Infolog: infoLog,
		ErrorLog: errorLog,
		Models: data.New(db),
		ErrorChan: make(chan error),
		ErrorChanDone: make(chan bool),
	}

	//setup do email
	app.Mailer = app.creteMail()
	//iniciar a escuta dos canais de email
	go app.listenForMail()

	//vericar o sinal de shutdown(terminar ou parar aplicacao)
	go app.ListenForShutdown()

	//verificar erros nos canais em background de envio de email de invoice
	go app.listenForError()

	app.serve()
}

func(app *Config) listenForError() {
	for {
		select {
		case err := <- app.ErrorChan:
			app.ErrorLog.Println(err)
		case <- app.ErrorChanDone:
			return
		}
	}
}

func(app *Config) serve() {
	srv := &http.Server{
		Addr: fmt.Sprintf(":%s", webPort),
		Handler: app.routes(),
	}

	app.Infolog.Println("Web server starting...")
	err := srv.ListenAndServe()
	if err != nil {
		log.Panic(err)
	}
}

func initDB() *sql.DB {
	conn := connectToDB()
	if conn == nil {
		log.Panic("cant connect to DB")
	}
	return conn
}

func connectToDB() *sql.DB{
	counts := 0

	dsn := os.Getenv("DSN")

	for {
		connection,err := openDB(dsn)
		if err != nil {
			log.Println("postgres not ready")
		} else {
			log.Println("connected to DB")
			return connection
		}

		if counts > 10 {
			return nil
		}

		log.Println("watting for 1 second to try again")
		time.Sleep(1 * time.Second)
		counts++
		continue
	}
}

func openDB(dsn string) (*sql.DB,error) {
	db,err := sql.Open("pgx", dsn)
	if err != nil {
		return nil,err
	}
	
	err = db.Ping()//testa conexao
	if err != nil {
		return nil,err
	}

	return db,nil
}

func initSession() *scs.SessionManager {
	gob.Register(data.User{})//para armazer na usuario na sessao
	session := scs.New()
	//informacoes da sessao sao armazenada no redis
	session.Store = redisstore.New(initRedis())
	session.Lifetime = 24 * time.Hour //sessao dura 24h
	session.Cookie.Persist = true
	session.Cookie.SameSite = http.SameSiteLaxMode
	session.Cookie.Secure = true

	return session
}

func initRedis() *redis.Pool {
	redisPool := &redis.Pool {
		MaxIdle: 10,//tempo maximo para conexao
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", os.Getenv("REDIS"))
		},
	}
	return redisPool
}

//shutdown para terminar as tarefas antes da aplicacao ser derrubada
func (app *Config) ListenForShutdown() {
	quit := make(chan os.Signal,1)
	//qnd tiver sinal de parar o app ou sinal de terminar o app envia noticacao para o canal
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	//blockeia ate ter o sinal de para ou terminar o app
	<-quit
	app.shutdown()//limpa as tarefas
	os.Exit(0)
}

func (app *Config) shutdown() {
	//limpa todas tarefas
	app.Infolog.Println("would run cleanup tasks...")

	//bloqueia o termino ate o wg(WaitGroup) estar vazio, ate enviar todos emails
	app.Wait.Wait()

	//indica que o envio de emails chegou ao fim
	app.Mailer.DoneChan <- true
	app.ErrorChanDone <- true

	app.Infolog.Println("closing channels and shutting down application...")
	close(app.Mailer.MailerChan)
	close(app.Mailer.ErrorChan)
	close(app.Mailer.DoneChan)
	close(app.ErrorChan)
	close(app.ErrorChanDone)
}

func (app *Config) creteMail() Mail{
	errChan := make(chan error)
	mailerChan := make(chan Message, 100)//buffer channel envia ate 100 msg simutaneas
	doneChan := make(chan bool)

	m := Mail{
		Domain: "localhost",
		Host: "localhost",
		Port: 1025,
		Encryption: "none",
		FromAddress: "info@mycompany.com",
		FromName: "test",
		Wait: app.Wait, //VERIFICAR
		ErrorChan: errChan,
		MailerChan: mailerChan,
		DoneChan : doneChan,
	}


	return m
}