package main

import (
	"log"
	"net/http"
	"os"

	"github.com/husio/surf"
	"github.com/husio/surf/csrf"
)

type configuration struct {
	HTTP   string
	Secret string
}

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(0)

	conf := configuration{
		HTTP:   env("HTTP", "127.0.0.1:8000"),
		Secret: env("SECRET", "asjdosaihdqohwd08hywqd0dwq8hoidahsaoihdqohdwqwqd9hgas"),
	}

	var store EntryStore

	tmpl := surf.LoadTemplates("./templates/*.tmpl")

	rt := surf.NewRouter()
	rt.Add("/", "GET", ListEntriesHandler(&store, tmpl))
	rt.Add("/entries/create", "GET,POST", CreateEntryHandler(&store, tmpl))
	rt.Add("/entries/(id)", "GET", ShowEntryHandler(&store, tmpl))

	app := csrf.Protect(rt, conf.Secret, tmpl)
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
