package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/go-surf/surf"
	"github.com/go-surf/surf/csrf"
	"github.com/go-surf/surf/jwt"
)

type configuration struct {
	HTTP     string
	Secret   string
	Database string
}

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(0)

	conf := configuration{
		HTTP:     env("HTTP", "127.0.0.1:8000"),
		Secret:   env("SECRET", "asjdosaihdqohwd08hywqd0dwq8hoidahsaoihdqohdwqwqd9hgas"),
		Database: env("DATABASE", ""),
	}

	var store EntryStore
	if conf.Database == "" {
		store = &MemoryEntryStore{}
		log.Println("using in memory storage")
	} else {
		if s, err := OpenSqliteEntryStore(conf.Database); err != nil {
			log.Fatalf("cannot connect to database: %s", err)
		} else {
			if err := s.Migrate(context.Background()); err != nil {
				log.Fatalf("cannot migrate sqlite: %s", err)
			}
			store = s
		}
	}

	signer := jwt.HMAC384(conf.Secret, "")

	tmpl := surf.LoadTemplates("./templates/*.tmpl")

	rt := surf.NewRouter()
	rt.Add("/", "GET", ListEntriesHandler(store, signer, tmpl))
	rt.Add("/entries/create", "GET,POST", CreateEntryHandler(store, signer, tmpl))
	rt.Add("/entries/(id)", "GET", ShowEntryHandler(store, tmpl))
	rt.Add("/entries/(id)/delete", "GET,POST", DeleteEntryHandler(store, signer, tmpl))

	rt.Add("/login", "GET", LoginHandler(signer, tmpl))
	rt.Add("/logout", "GET", LogoutHandler())

	flashEngine := surf.NewFlashCookieEngine(signer)

	app := surf.WithRequestID(
		surf.WithLogging(
			csrf.Protect(conf.Secret, tmpl,
				surf.WithFlashMessages(flashEngine, rt))))
	if err := http.ListenAndServe(conf.HTTP, app); err != nil {
		log.Fatalf("http server: %v", err)
	}
}

func env(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}
