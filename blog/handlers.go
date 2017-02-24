package main

import (
	"html/template"
	"log"
	"net/http"

	"github.com/husio/surf"
	"github.com/husio/surf/csrf"
)

func ListEntriesHandler(
	store *EntryStore,
	rend surf.Renderer,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entries, err := store.Latest(3)
		if err != nil {
			log.Fatalf("cannot get latest entries: %s", err)
			rend.RenderDefault(w, http.StatusInternalServerError)
			return
		}

		rend.Render(w, http.StatusOK, "list.tmpl", struct {
			Entries []Entry
		}{
			Entries: entries,
		})
	}
}

func CreateEntryHandler(
	store *EntryStore,
	rend surf.Renderer,
) http.HandlerFunc {

	form := surf.NewForm(
		surf.NewTextField("title").MinLen(3).MaxLen(32).Required().Autofocus().Autocomplete("off"),
		surf.NewTextareaField("content").Required().Label("Blog entry content"),
	)

	type content struct {
		Error     error
		Fields    surf.RenderedFields
		CsrfField template.HTML
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			if err := r.ParseForm(); err != nil {
				rend.Render(w, http.StatusBadRequest, "create.tmpl", content{
					Error:     err,
					Fields:    form.Render(r),
					CsrfField: csrf.Tag(r),
				})
				return
			}
			if form.IsValid(r) {
				entry, err := store.Create(r.Form.Get("title"), r.Form.Get("content"))
				if err != nil {
					log.Printf("cannot create entry: %s", err)
					rend.RenderDefault(w, http.StatusInternalServerError)
					return
				}
				http.Redirect(w, r, "/entries/"+entry.ID, http.StatusSeeOther)
				return
			}
		}

		code := http.StatusOK
		if r.Method == "POST" {
			code = http.StatusBadRequest
		}
		rend.Render(w, code, "create.tmpl", content{
			Fields:    form.Render(r),
			CsrfField: csrf.Tag(r),
		})
	}
}

func ShowEntryHandler(
	store *EntryStore,
	rend surf.Renderer,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entry, err := store.ByID(surf.PathArg(r, 0))
		switch err {
		case nil:
			// all good
		case ErrNotFound:
			rend.RenderDefault(w, http.StatusNotFound)
			return
		default:
			log.Printf("cannot get entry by id: %s", err)
			rend.RenderDefault(w, http.StatusInternalServerError)
			return
		}

		rend.Render(w, http.StatusOK, "show.tmpl", struct {
			Entry *Entry
		}{
			Entry: entry,
		})
	}
}
