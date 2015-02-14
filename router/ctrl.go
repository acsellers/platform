package router

import (
	"fmt"
	"log"
	"net/http"
	"reflect"

	"golang.org/x/net/websocket"
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
	if adc, ok := ctrl.(autoDupeCtrl); ok {
		return adc.Name()
	}
	return reflect.TypeOf(ctrl).Name()
}

func callCtrl(w http.ResponseWriter, r *http.Request, l Leaf, p map[string]string, lg *log.Logger) (Controller, Result) {
	ctrl := l.Ctrl.Dupe()
	ctrl.SetRequestData(w, r)
	ctrl.SetID(p)
	ctrl.SetLogger(lg)
	name := ctrlName(ctrl)
	lg.Printf("Starting request for %s using %s.%s\n", r.URL.String(), name, l.Action)
	if l.SetContext {
		if sc, ok := ctrl.(contexter); ok {
			sc.SetContext(map[string]interface{}{})
		}
	}
	if l.PreFilter {
		if pf, ok := ctrl.(prefilter); ok {
			lg.Println("Running PreFilter")
			res := pf.PreFilter()
			if res != nil {
				lg.Printf("PreFilter returned %s\n", res.String())
				return nil, res
			}
		}
	}
	if l.PreItem && l.Item {
		if pi, ok := ctrl.(preitem); ok {
			lg.Println("Running PreItem")
			res := pi.PreItem()
			if res != nil {
				lg.Printf("PreItem returned %s\n", res.String())
				return nil, res
			}
		}
	}

	return ctrl, nil
}

type BaseController struct {
	http.ResponseWriter `dupe:"no"`
	Request             *http.Request `dupe:"no"`
	Log                 *log.Logger   `dupe:"no"`
	// The Cache is shared between all Controllers
	Cache map[string]interface{}
	// The Context will be a new map each request
	Context map[string]interface{}
	// URL Params
	ID map[string]string
}

func (bc *BaseController) SetRequestData(w http.ResponseWriter, r *http.Request) {
	bc.ResponseWriter = w
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

func (bc *BaseController) SetID(p map[string]string) {
	bc.ID = p
}

type Controller interface {
	Path() string
	SetRequestData(http.ResponseWriter, *http.Request)
	SetID(map[string]string)
	SetLogger(*log.Logger)
}

type DupableController interface {
	Controller
	Dupe() Controller
}

type nullCtrl struct{}

func (nullCtrl) Path() string {
	return ""
}
func (nullCtrl) Dupe() Controller {
	return nullCtrl{}
}
func (nullCtrl) SetLogger(*log.Logger) {
}
func (nullCtrl) SetRequestData(http.ResponseWriter, *http.Request) {
}
func (nullCtrl) SetID(map[string]string) {
}

type ctrlHF struct {
	handler http.HandlerFunc
	w       http.ResponseWriter
	r       *http.Request
}

func (ctrlHF) Path() string {
	return ""
}
func (c ctrlHF) Dupe() Controller {
	return &ctrlHF{handler: c.handler}
}
func (ctrlHF) SetLogger(*log.Logger) {
}
func (c *ctrlHF) SetRequestData(w http.ResponseWriter, r *http.Request) {
	c.w = w
	c.r = r
}
func (ctrlHF) SetID(map[string]string) {
}

func wshandler(wf WSHandlerFunc) func(http.ResponseWriter, *http.Request) {
	h := websocket.Handler(wf)
	return func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)
	}
}

type autoDupeCtrl struct {
	Controller
}

func (adc autoDupeCtrl) Name() string {
	return reflect.TypeOf(adc.Controller).Name()
}
func (adc autoDupeCtrl) Dupe() Controller {
	rs := reflect.ValueOf(adc.Controller)
	rt := reflect.TypeOf(adc.Controller)
	rn := reflect.New(rt)
	for i := 0; i < rt.NumField(); i++ {
		switch rn.Elem().Field(i).Kind() {
		case reflect.Struct:
			if rt.Field(i).Tag.Get("dupe") != "no" {
				reflectDupe(rs.Field(i), rn.Elem().Field(i).Addr())
			} else {
				rn.Elem().Field(i).Set(rs.Field(i))
			}
		case reflect.Ptr:
			if rt.Field(i).Type.Elem().Kind() == reflect.Struct {
				if rt.Field(i).Tag.Get("dupe") != "no" {
					ov := reflect.New(rt.Field(i).Type.Elem())
					reflectDupe(rs.Field(i).Elem(), ov)
					rn.Elem().Field(i).Set(ov)
				} else {
					rn.Elem().Field(i).Set(rs.Field(i))
				}
			} else {
				rn.Elem().Field(i).Set(rs.Field(i))
			}
		default:
			if rs.Field(i).IsValid() && rn.Elem().Field(i).CanSet() {
				rn.Elem().Field(i).Set(rs.Field(i))
			}
		}
	}
	return rn.Interface().(Controller)
}

func reflectDupe(sv, ov reflect.Value) {
	if !sv.IsValid() {
		return
	}
	rt := sv.Type()
	for i := 0; i < rt.NumField(); i++ {
		switch sv.Field(i).Kind() {
		case reflect.Struct:
			if rt.Field(i).Tag.Get("dupe") != "no" {
				reflectDupe(sv.Field(i), ov.Elem().Field(i).Addr())
			} else {
				ov.Elem().Field(i).Set(sv.Field(i))
			}
		case reflect.Ptr:
			if rt.Field(i).Type.Elem().Kind() == reflect.Struct {
				if rt.Field(i).Tag.Get("dupe") != "no" {
					oov := reflect.New(rt.Field(i).Type.Elem())
					reflectDupe(sv.Field(i).Elem(), oov)
					ov.Elem().Field(i).Set(oov)
				} else {
					ov.Elem().Field(i).Set(sv.Field(i))
				}
			} else {
				ov.Elem().Field(i).Set(sv.Field(i))
			}
		default:
			ov.Elem().Field(i).Set(sv.Field(i))
		}
	}
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
	WSItem(*websocket.Conn)

	// MultiCtrl only
	New() Result
	Create() Result
	Index() Result
	WSBase(*websocket.Conn)

	// Extra Action Defintion functions
	OtherBase(*SubRoute)
	OtherItem(*SubRoute)
}
