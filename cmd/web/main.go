package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/alexedwards/scs/redisstore"
	"github.com/alexedwards/scs/v2"
	"github.com/gomodule/redigo/redis"
	_ "github.com/jackc/pgconn"
	_ "github.com/jackc/pgx/v4"
	_ "github.com/jackc/pgx/v4/stdlib"
)

var webPort = "8000"

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
	}

	app.serve()
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