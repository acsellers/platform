package router

import (
	"fmt"
	"log"
	"net/http"
	"reflect"
)

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
}

type DupableController interface {
	Controller
	Dupe() Controller
}

type autoDupeCtrl struct {
	Controller
}

func (adc autoDupeCtrl) Dupe() Controller {
	rs := reflect.ValueOf(adc.Controller)
	rt := reflect.TypeOf(adc.Controller)
	rn := reflect.New(rt)
	for i := 0; i < rt.NumField(); i++ {
		rn.Elem().Field(i).Set(rs.Field(i))
	}
	return rn.Interface().(Controller)
}

// RestfulController lists all the possible functions
// that may be implemented by controllers, note that you
// should implement a subset of the functions as needed.
// The only requred functions for a Controller are in the
// Controller, which are handled by BaseController.
type RestfulController interface {
	// Run before all requests if present
	PreFilter() Result
	// Run before requests with an item if present
	PreItem() Result

	// SingleCtrl & MultiCtrl
	Show() Result
	Edit() Result
	Update() Result
	Delete() Result

	// MultiCtrl only
	New() Result
	Create() Result
	Index() Result
}

type ResetController struct{}

// SingleCtrl & MultiCtrl
func (r ResetController) Show() Result {
	return NotFound{}
}
func (r ResetController) Edit() Result {
	return NotFound{}
}
func (r ResetController) Update() Result {
	return NotFound{}
}
func (r ResetController) Delete() Result {
	return NotFound{}
}

// MultiCtrl only
func (r ResetController) New() Result {
	return NotFound{}
}
func (r ResetController) Create() Result {
	return NotFound{}
}
func (r ResetController) Index() Result {
	return NotFound{}
}

type beenReset interface {
	resetFunc(string, string) bool
}

// The reason resetController works, kind of a hack around func == func
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
