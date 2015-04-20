package router

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/net/websocket"
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
	if r.Status != 0 {
		w.WriteHeader(r.Status)
	}
	io.Copy(w, r.Content)
}

func (r Rendered) String() string {
	return "Rendered Data"
}

func JSON(data interface{}) Result {
	return JSONData{Data: data}
}

type JSONData struct {
	Data   interface{}
	Status int
}

func (JSONData) SetRequest(*http.Request) {
}

func (r JSONData) Execute(w http.ResponseWriter) {
	if r.Status != 0 {
		w.WriteHeader(r.Status)
	}
	json.NewEncoder(w).Encode(r.Data)
}

func (r JSONData) String() string {
	return "JSON Data"
}

func JSONP(data interface{}) Result {
	return &JSONPData{Data: data}
}

type JSONPData struct {
	Data    interface{}
	Status  int
	Request *http.Request
}

func (r *JSONPData) SetRequest(req *http.Request) {
	r.Request = req
}

func (r JSONPData) Execute(w http.ResponseWriter) {
	if r.Status != 0 {
		w.WriteHeader(r.Status)
	}
	io.WriteString(w, r.Request.URL.Query().Get("callback")+"(")
	json.NewEncoder(w).Encode(r.Data)
	io.WriteString(w, ")")
}

func (r JSONPData) String() string {
	return "JSONP Data"
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

func RedirectTo(location string) Result {
	return &Redirect{URL: location}
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

func Disallow(location string) Result {
	return &NotAllowed{Fallback: location}
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
	} else if na.Content != nil {
		if na.Status != 0 {
			w.WriteHeader(na.Status)
		}
		io.Copy(w, na.Content)
	} else if na.Status != 0 {
		w.WriteHeader(na.Status)
	} else {
		w.WriteHeader(403)
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

type WSResult struct {
	Handler websocket.Handler
	request *http.Request
}

func (ws *WSResult) SetRequest(r *http.Request) {
	ws.request = r
}
func (ws WSResult) Execute(w http.ResponseWriter) {
	ws.Handler.ServeHTTP(w, ws.request)
}

func (WSResult) String() string {
	return "Websocket Connection"
}

type UniqueHandler struct {
	Handler http.Handler
	request *http.Request
}

func (uh *UniqueHandler) SetRequest(r *http.Request) {
	uh.request = r
}

func (uh UniqueHandler) Execute(w http.ResponseWriter) {
	go uh.Handler.ServeHTTP(w, uh.request)
}

func (UniqueHandler) String() string {
	return "One-off Handler"
}
