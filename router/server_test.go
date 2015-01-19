package router

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type restCtrl struct {
	Loc string
	*BaseController
}

func (r restCtrl) Path() string {
	return r.Loc
}

func (r restCtrl) Index() Result {
	return Rendered{
		Content: strings.NewReader("Index"),
	}
}

func (r restCtrl) Show() Result {
	return Rendered{
		Content: strings.NewReader("Show: " + r.Params[":"+r.Loc+"id"]),
	}
}
func (r restCtrl) Hello() Result {
	return Rendered{
		Content: strings.NewReader("Hello: " + r.Params[":"+r.Loc+"id"]),
	}
}
func (r restCtrl) Bye() Result {
	return Rendered{
		Content: strings.NewReader("Goodbye"),
	}

}
func (r restCtrl) OtherBase(sr *SubRoute) {
	sr.Get("all").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "all")
	})
	sr.Get("bye").Action("Bye")
}

func (r restCtrl) OtherItem(sr *SubRoute) {
	sr.Get("asdf").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "asdf")
	})
	sr.Get("hello").Action("Hello")
}

func TestRestControllers(t *testing.T) {
	r := NewRouter()
	r.LogOutput = ioutil.Discard
	r.Many(restCtrl{"posts", &BaseController{}})
	s := httptest.NewServer(r)
	defer s.Close()

	ir, err := http.Get(s.URL + "/posts")
	if err != nil {
		t.Fatal("GET Index:", err)
	}
	defer ir.Body.Close()
	body, err := ioutil.ReadAll(ir.Body)
	if string(body) != "Index" {
		t.Fatal("Unexpected Response, expected 'Index' got:", body)
	}

	ir, err = http.Get(s.URL + "/posts/123")
	if err != nil {
		t.Fatal("GET Show:", err)
	}
	defer ir.Body.Close()
	body, err = ioutil.ReadAll(ir.Body)
	if string(body) != "Show: 123" {
		t.Fatal("Unexpected Response, expected 'Show: 123' got:", body)
	}

	ir, err = http.Get(s.URL + "/posts/123/asdf")
	if err != nil {
		t.Fatal("GET OtherItem (asdf):", err)
	}
	defer ir.Body.Close()
	body, err = ioutil.ReadAll(ir.Body)
	if string(body) != "asdf" {
		t.Fatal("Unexpected Response, expected 'asdf' got:", body)
	}

	ir, err = http.Get(s.URL + "/posts/123/hello")
	if err != nil {
		t.Fatal("GET OtherItem (hello):", err)
	}
	defer ir.Body.Close()
	body, err = ioutil.ReadAll(ir.Body)
	if string(body) != "Hello: 123" {
		t.Fatal("Unexpected Response, expected 'Hello: 123' got:", body)
	}

	ir, err = http.Get(s.URL + "/posts/all")
	if err != nil {
		t.Fatal("GET OtherBase (all):", err)
	}
	defer ir.Body.Close()
	body, err = ioutil.ReadAll(ir.Body)
	if string(body) != "all" {
		t.Fatal("Unexpected Response, expected 'all' got:", body)
	}

	ir, err = http.Get(s.URL + "/posts/bye")
	if err != nil {
		t.Fatal("GET OtherBase (bye):", err)
	}
	defer ir.Body.Close()
	body, err = ioutil.ReadAll(ir.Body)
	if string(body) != "Goodbye" {
		t.Fatal("Unexpected Response, expected 'Goodbye' got:", body)
	}
}
