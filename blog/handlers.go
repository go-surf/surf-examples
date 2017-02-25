package main

import (
	"html/template"
	"log"
	"net/http"

	"github.com/go-surf/surf"
	"github.com/go-surf/surf/csrf"
)

func ListEntriesHandler(
	store *EntryStore,
	rend surf.Renderer,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entries, err := store.Latest(r.Context(), 20)
		if err != nil {
			log.Fatalf("cannot get latest entries: %s", err)
			rend.RenderDefault(w, http.StatusInternalServerError)
			return
		}

		rend.Render(w, http.StatusOK, "list.tmpl", struct {
			FlashMessages []surf.FlashMessage
			Entries       []Entry
		}{
			FlashMessages: surf.ConsumeFlashMessages(r.Context()),
			Entries:       entries,
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
					CsrfField: csrf.FormField(r),
				})
				return
			}
			if form.IsValid(r) {
				entry, err := store.Create(r.Context(),
					r.Form.Get("title"), r.Form.Get("content"))
				if err != nil {
					log.Printf("cannot create entry: %s", err)
					rend.RenderDefault(w, http.StatusInternalServerError)
					return
				}

				surf.AddFlashMessage(r.Context(), "success", "Blog entry created")
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
			CsrfField: csrf.FormField(r),
		})
	}
}

func ShowEntryHandler(
	store *EntryStore,
	rend surf.Renderer,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entry, err := store.ByID(r.Context(), surf.PathArg(r, 0))
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
			FlashMessages []surf.FlashMessage
			Entry         *Entry
		}{
			FlashMessages: surf.ConsumeFlashMessages(r.Context()),
			Entry:         entry,
		})
	}
}

func DeleteEntryHandler(
	store *EntryStore,
	rend surf.Renderer,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			entry, err := store.ByID(r.Context(), surf.PathArg(r, 0))
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
			rend.Render(w, http.StatusOK, "delete.tmpl", struct {
				CsrfField template.HTML
				Entry     *Entry
			}{
				CsrfField: csrf.FormField(r),
				Entry:     entry,
			})
			return
		}

		switch err := store.Delete(r.Context(), surf.PathArg(r, 0)); err {
		case nil:
			surf.AddFlashMessage(r.Context(), "info", "Blog entry deleted")
			http.Redirect(w, r, "/", http.StatusSeeOther)
		case ErrNotFound:
			rend.RenderDefault(w, http.StatusNotFound)
		default:
			log.Printf("cannot delete entry: %s", err)
			rend.RenderDefault(w, http.StatusInternalServerError)
		}
	}
}
