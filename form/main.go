package main

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"regexp"

	"github.com/go-surf/surf"
)

func main() {
	http.HandleFunc("/", handleForm)
	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatalf("http server: %s", err)
	}
}

func handleForm(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	err := testtmpl.Execute(w, struct {
		Valid  bool
		Fields surf.RenderedFields
		Values url.Values
	}{
		Valid:  userForm.IsValid(r),
		Fields: userForm.Render(r),
		Values: r.Form,
	})

	if err != nil {
		fmt.Fprintf(w, "Error: %#v", err)
	}

}

var userForm = surf.NewForm(
	surf.NewTextField("user-name").Required().MinLen(3).MaxLen(22).Autofocus().Placeholder("Full name").Pass(FullName),
	surf.NewNumberField("user_age").Min(12).Max(99).Default(22).Required(),
	surf.NewSelectField("hobby", []surf.SelectFieldChoice{
		{Value: "cs", Label: "Collecting Seashells"},
		{Value: "css", Label: "Collecting Stamps"},
		{Value: "g", Label: "Gardening"},
	}).Multiple(),
	surf.NewCheckboxField("terms-and-conditions").Label("Accept my terms!").Default(true).Required(),
)

func FullName(name string) error {
	if ok, _ := regexp.MatchString(`^\w+ \w+$`, name); !ok {
		return errors.New("invalid name")
	}
	return nil
}

var testtmpl = template.Must(template.New("").Parse(`
<!doctype html>
<link href="//maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap.min.css" rel="stylesheet">
<div class="container">
	{{if .Values}}
	<p>
		Submitted values: <code>{{.Values}}</code>
	</p>
	<hr>
	{{end}}

	<div class="alert alert-{{if .Valid}}success{{else}}warning{{end}}">
		{{if .Valid}}Success!{{else}}Correct mistakes.{{end}}
	</div>

	<div class="row">
		<form action="." method="POST">
			{{.Fields.RenderBootstrap}}
			<button class="btn" type="submit">Submit</button>
		</form>
	</div>
</div>
`))
