package router

import (
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

}
