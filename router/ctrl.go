package router

import (
	"fmt"
	"log"
	"net/http"
	"reflect"
)

type NotApplicable struct{}

func (na NotApplicable) Error() string {
	return "Request Not Applicable"
}

// RedirectError may be returned by PreFilter, PreItem or any
// Restful function and is intended for use for authentication
// checks or permission checks.
type RedirectError struct {
	Reason   string
	Location string
	Code     int
}

func (re RedirectError) Error() string {
	return fmt.Sprintf("Redirect to '%s' for: %s", re.Location, re.Reason)
}
func (re RedirectError) Redirect(w http.ResponseWriter, r *http.Request) {
	if re.Code == 0 {
		re.Code = 302
	}
	http.Redirect(w, r, re.Location, re.Code)
}

func ctrlName(ctrl interface{}) string {
	return reflect.TypeOf(ctrl).Name()
}

func callCtrl(w http.ResponseWriter, r *http.Request, l Leaf, p map[string]string, lg *log.Logger) (Controller, Result) {
	ctrl := l.Ctrl.Dupe()
	ctrl.SetRequestData(w, r)
	ctrl.SetParams(p)
	ctrl.SetLogger(lg)
	name := ctrlName(ctrl)
	lg.Printf("Starting request for %s using %s.%s\n", r.URL.String(), name, l.Action)
	if pf, ok := ctrl.(interface {
		PreFilter() Result
	}); ok {
		lg.Println("Running PreFilter")
		res := pf.PreFilter()
		if res != nil {
			lg.Printf("PreFilter returned %s\n", res.String())
			return nil, res
		}
	}
	if pi, ok := ctrl.(interface {
		PreItem() Result
	}); ok && l.Item {
		lg.Println("Running PreItem")
		res := pi.PreItem()
		if res != nil {
			lg.Printf("PreItem returned %s\n", res.String())
			return nil, res
		}
	}

	return ctrl, nil
}

type BaseController struct {
	Out     http.ResponseWriter
	Request *http.Request
	Log     *log.Logger
	// The Cache is shared between all Controllers
	Cache map[string]interface{}
	// The Context will be a new map each request
	Context map[string]interface{}
	// URL Params
	Params map[string]string
}

func (bc *BaseController) SetRequestData(w http.ResponseWriter, r *http.Request) {
	bc.Out = w
	bc.Request = r
}

func (bc *BaseController) SetLogger(l *log.Logger) {
	bc.Log = l
}

func (bc *BaseController) SetCache(c map[string]interface{}) {
	bc.Cache = c
}

func (bc *BaseController) SetContext(c map[string]interface{}) {
	bc.Context = c
}

func (bc *BaseController) SetParams(p map[string]string) {
	bc.Params = p
}

type Controller interface {
	Path() string
	SetRequestData(http.ResponseWriter, *http.Request)
	SetParams(map[string]string)
	SetLogger(*log.Logger)
	Dupe() Controller
}

// RestfulController lists all the possible functions
// that may be implemented by controllers, note that you
// should implement a subset of the functions as needed.
// The only requred functions for a Controller are in the
// Controller, which are handled by BaseController.
type RestfulController interface {
	// Run before all requests if present
	PreFilter() error
	// Run before requests with an item if present
	PreItem() error

	// SingleCtrl & MultiCtrl
	Show() error
	Edit() error
	Update() error
	Delete() error

	// MultiCtrl only
	New() error
	Create() error
	Index() error

	// Non-Restful Routes
	Member() map[string]Member
	// Collection is only mapped on Many Controllers
	Collection() map[string]Collection
}

type ResetController struct{}

// SingleCtrl & MultiCtrl
func (r ResetController) Show() error {
	return NotApplicable{}
}
func (r ResetController) Edit() error {
	return NotApplicable{}
}
func (r ResetController) Update() error {
	return NotApplicable{}
}
func (r ResetController) Delete() error {
	return NotApplicable{}
}

// MultiCtrl only
func (r ResetController) New() error {
	return NotApplicable{}
}
func (r ResetController) Create() error {
	return NotApplicable{}
}
func (r ResetController) Index() error {
	return NotApplicable{}
}

// Non-Restful Routes
func (r ResetController) Member() map[string]Member {
	return nil
}
func (r ResetController) Collection() map[string]Collection {
	return nil
}

type beenReset interface {
	resetFunc(string, string) bool
}

// Secret Function to be awesome
func (r ResetController) resetFunc(name, fp string) bool {
	switch name {
	case "Show":
		return fp == fmt.Sprint(r.Show)
	case "Edit":
		return fp == fmt.Sprint(r.Edit)
	case "Update":
		return fp == fmt.Sprint(r.Update)
	case "Delete":
		return fp == fmt.Sprint(r.Delete)
	case "New":
		return fp == fmt.Sprint(r.New)
	case "Create":
		return fp == fmt.Sprint(r.Create)
	case "Index":
		return fp == fmt.Sprint(r.Index)
	}
	return false
}

type Member struct {
	Methods  []string
	Path     string
	Callable func(Controller) error
}

type Collection struct {
	Methods  []string
	Path     string
	Callable func(Controller) error
}
