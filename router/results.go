package router

import (
	"fmt"
	"io"
	"net/http"
)

type Result interface {
	SetRequest(*http.Request)
	Execute(http.ResponseWriter)
	String() string
}

type Rendered struct {
	Content io.Reader
	Status  int
}

func (Rendered) SetRequest(*http.Request) {
}

func (r Rendered) Execute(w http.ResponseWriter) {
	io.Copy(w, r.Content)
}

func (r Rendered) String() string {
	return "Rendered Data"
}

type String struct {
	Content string
	Status  int
}

func (String) SetRequest(*http.Request) {
}
func (r String) Execute(w http.ResponseWriter) {
	io.WriteString(w, r.Content)
}

func (r String) String() string {
	return "String Data"
}

type Redirect struct {
	Request *http.Request
	URL     string
	Status  int
}

func (r *Redirect) SetRequest(req *http.Request) {
	r.Request = req
}
func (r Redirect) Execute(w http.ResponseWriter) {
	if r.Status == 0 {
		r.Status = 303
	}
	http.Redirect(w, r.Request, r.URL, r.Status)
}

func (r Redirect) String() string {
	return fmt.Sprintf("Redirect To %s", r.URL)
}

type NotAllowed struct {
	Request  *http.Request
	Content  io.Reader
	Fallback string
	Status   int
}

func (na *NotAllowed) SetRequest(req *http.Request) {
	na.Request = req
}
func (na NotAllowed) Execute(w http.ResponseWriter) {
	if na.Fallback != "" {
		if na.Status == 0 {
			na.Status = 302
		}
		http.Redirect(w, na.Request, na.Fallback, na.Status)
	} else {
		if na.Status != 0 {
			w.WriteHeader(na.Status)
		}
		io.Copy(w, na.Content)
	}
}

func (na NotAllowed) String() string {
	if na.Fallback != "" {
		return fmt.Sprintf("Not Allowed, Redirecting To %s", na.Fallback)
	} else {
		return fmt.Sprintf("Not Allowed, Rendered a Response")
	}
}

type InternalError struct {
	Error error
}

func (InternalError) SetRequest(*http.Request) {
}
func (ie InternalError) Execute(w http.ResponseWriter) {
	io.WriteString(w, "<h1>Internal Server Error</h1>")
}

func (ie InternalError) String() string {
	return fmt.Sprintf("Error: %s", ie.Error.Error())
}

type NothingResult struct{}

func (NothingResult) SetRequest(*http.Request) {
}
func (NothingResult) Execute(http.ResponseWriter) {
}

func (NothingResult) String() string {
	return "Being Handled Elsewhere"
}

type NotFound struct{}

func (NotFound) SetRequest(*http.Request) {
}
func (NotFound) Execute(http.ResponseWriter) {
}

func (NotFound) String() string {
	return "NotFound/NotApplicable"
}
