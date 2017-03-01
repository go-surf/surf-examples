package main

import (
	"html/template"
	"log"
	"net/http"

	"github.com/go-surf/surf"
	"github.com/go-surf/surf/csrf"
)

func ListEntriesHandler(
	store EntryStore,
	verifier surf.Verifier,
	rend surf.Renderer,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user User
		surf.CurrentAccount(r, verifier, &user)

		entries, err := store.Latest(r.Context(), 20)
		if err != nil {
			log.Fatalf("cannot get latest entries: %s", err)
			rend.RenderDefault(w, http.StatusInternalServerError)
			return
		}

		rend.Render(w, http.StatusOK, "list.tmpl", struct {
			FlashMessages []surf.FlashMessage
			Entries       []Entry
			User          *User
		}{
			FlashMessages: surf.ConsumeFlashMessages(r.Context()),
			Entries:       entries,
			User:          &user,
		})
	}
}

func CreateEntryHandler(
	store EntryStore,
	verifier surf.Verifier,
	rend surf.Renderer,
) http.HandlerFunc {

	form := surf.NewForm(
		surf.NewTextField("title").MinLen(3).MaxLen(32).Required().Autofocus().Autocomplete("off"),
		surf.NewTextareaField("content").Required().Label("Blog entry content"),
	)

	type content struct {
		Error     error
		Fields    surf.RenderedFields
		User      *User
		CsrfField template.HTML
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var user User
		if !surf.CurrentAccount(r, verifier, &user) {
			rend.RenderDefault(w, http.StatusUnauthorized)
			return
		}

		if r.Method == "POST" {
			if err := r.ParseForm(); err != nil {
				rend.Render(w, http.StatusBadRequest, "create.tmpl", content{
					Error:     err,
					Fields:    form.Render(r),
					User:      &user,
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
			User:      &user,
			CsrfField: csrf.FormField(r),
		})
	}
}

func ShowEntryHandler(
	store EntryStore,
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
	store EntryStore,
	verifier surf.Verifier,
	rend surf.Renderer,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !surf.CurrentAccount(r, verifier, &User{}) {
			rend.RenderDefault(w, http.StatusUnauthorized)
			return
		}

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

func LoginHandler(
	signer surf.Signer,
	rend surf.Renderer,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if surf.CurrentAccount(r, signer, &User{}) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		surf.Login(w, signer, &User{ID: 5, Name: "Bob"})
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func LogoutHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		surf.Logout(w)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

type User struct {
	ID   int
	Name string
}

func (u *User) Authenticated() bool {
	return u.ID != 0
}
