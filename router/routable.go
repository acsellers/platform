package router

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

	"golang.org/x/net/websocket"
)

type Router struct {
	Tree      *RetrieveTree
	cache     map[string]interface{}
	OnError   func(error, http.ResponseWriter, *http.Request, Controller)
	LogOutput io.Writer
}

func NewRouter() *Router {
	r := &Router{Tree: NewTree()}
	r.cache = make(map[string]interface{})
	r.LogOutput = os.Stdout
	return r
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	logBuffer := &bytes.Buffer{}
	io.WriteString(logBuffer, "\n\n")
	reqLog := log.New(logBuffer, "", log.Lmicroseconds)
	defer io.Copy(r.LogOutput, logBuffer)
	now := time.Now()

	results := r.Tree.RetrieveWithFallback(req.URL.Path)
	reqLog.Printf("%d Possible Handlers, %d Fallback Handlers", len(results.Primary), len(results.Secondary))
	if len(results.Primary) == 0 && len(results.Secondary) == 0 {
		if results.Fallback == nil {
			reqLog.Println("Bad Route", req.URL.Path)
			// r.BadRoute(w, req)
			return
		} else {
			results.Fallback.ServeHTTP(w, req)
		}
	}

	for _, handler := range append(results.Primary, results.Secondary...) {
		schemeMatch := false
		if req.Header.Get("Upgrade") == "websocket" {
			schemeMatch = handler.Scheme == "ws"
		} else if handler.Scheme == "http" {
			schemeMatch = true
		}
		if handler.Method != req.Method || !schemeMatch {
			reqLog.Printf("Skipping %s.%s due to incorrect method\n", ctrlName(handler.Ctrl), handler.Action)
			continue
		}
		// prepare
		ctrl, res := callCtrl(w, req, handler, results.ID, reqLog)
		if res != nil {
			if _, ok := res.(NotFound); ok {
				reqLog.Println("Aborting current handler, starting next handler")
				continue
			}
			res.SetRequest(req)
			res.Execute(w)
			reqLog.Println(res)
			reqLog.Printf("Completed request in %v\n", time.Since(now))
			return
		}
		res = handler.Callable(ctrl)
		if res == nil {
			continue
		}
		res.SetRequest(req)
		res.Execute(w)
		reqLog.Println(res)
		reqLog.Printf("Completed request in %v\n", time.Since(now))
		return
	}
	if results.Fallback != nil {
		results.Fallback.ServeHTTP(w, req)
	} else {
		http.NotFound(w, req)
	}
	reqLog.Printf("Completed request in %v\n", time.Since(now))
}

func (r *Router) One(ctrl Controller) *SubRoute {
	sr := SubRoute{local: r.Tree.Branch}
	return sr.One(ctrl)
}

func (r *Router) Many(ctrl Controller) *SubRoute {
	sr := SubRoute{local: r.Tree.Branch}
	return sr.Many(ctrl)
}

// Namespace creates a
func (r *Router) Namespace(name string) *SubRoute {
	sr := SubRoute{local: r.Tree.Branch}
	return sr.Namespace(name)
}

// SetHandler sets an http.Handler to be called when there isn't a match
// for other SubRoutes. This is the base handler, best used for a top-level
// 404 Handler.
func (r *Router) SetHandler(h http.Handler) {
	r.Tree.Branch.Fallback = h
}

func (r *Router) PrefixHandler(prefix string, h http.Handler) *SubRoute {
	sr := r.Namespace(prefix)
	sr.SetHandler(h)
	return sr
}

type Module interface {
	Load(sr *SubRoute)
}

func (r *Router) Mount(m Module) {
	m.Load(&SubRoute{local: r.Tree.Branch})
}

type RouteDesc struct {
	Name   string
	Method string
	Path   string
}

func (r *Router) RouteList() []RouteDesc {
	rl := r.Tree.ListLeaves()
	rds := make([]RouteDesc, len(rl))
	for i := range rl {
		rds[i] = RouteDesc{
			Name:   rl[i].Name,
			Method: rl[i].Method,
			Path:   rl[i].Path,
		}
	}

	return rds
}

type SubRoute struct {
	local *Branch
	name  string
	ctrl  DupableController
}

type showController interface {
	Show() Result
}
type editController interface {
	Edit() Result
}
type updateController interface {
	Update() Result
}
type deleteController interface {
	Delete() Result
}
type newController interface {
	New() Result
}
type createController interface {
	Create() Result
}
type indexController interface {
	Index() Result
}
type otherBaseController interface {
	OtherBase(*SubRoute)
}
type otherItemController interface {
	OtherItem(sr *SubRoute)
}
type wsItemController interface {
	WSItem(*websocket.Conn)
}
type wsBaseController interface {
	WSBase(*websocket.Conn)
}

func (sr *SubRoute) One(ctrl Controller) *SubRoute {
	var dc DupableController
	var ok bool
	if dc, ok = ctrl.(DupableController); !ok {
		dc = autoDupeCtrl{ctrl}
	}

	name := ctrl.Path()
	urlname := name
	if len(sr.name) > 0 {
		urlname = name + "_" + sr.name
	}

	sr.insertShow(dc, ctrl, name, urlname+"_path", false)
	sr.insertEdit(dc, ctrl, name+"/edit", "edit_"+urlname+"_path", false)
	sr.insertUpdate(dc, ctrl, name, "update_"+urlname+"_path", false)
	sr.insertDelete(dc, ctrl, name, "delete_"+urlname+"_path", false)
	sr.insertOtherBase(dc, ctrl, urlname)
	sr.insertOtherItem(dc, ctrl, name)
	sr.insertWSItem(dc, ctrl, name, urlname+"_path", false)

	return &SubRoute{local: sr.local.InsertPath(name)}
}

func (sr *SubRoute) Many(ctrl Controller) *SubRoute {
	var dc DupableController
	var ok bool
	if dc, ok = ctrl.(DupableController); !ok {
		dc = autoDupeCtrl{ctrl}
	}
	name := ctrl.Path()
	itemName := fmt.Sprintf("%[1]s/:%[1]s", name)
	urlname := name
	if len(sr.name) > 0 {
		urlname = name + "_" + sr.name
	}

	sr.insertShow(dc, ctrl, itemName, "show_"+urlname+"_path", true)
	sr.insertEdit(dc, ctrl, itemName+"/edit", "edit_"+urlname+"_path", true)
	sr.insertUpdate(dc, ctrl, itemName, "update_"+urlname+"_path", true)
	sr.insertDelete(dc, ctrl, itemName, "delete_"+urlname+"_path", true)

	sr.insertNew(dc, ctrl, name+"/new", "new_"+urlname+"_path", false)
	sr.insertCreate(dc, ctrl, name, "create_"+urlname+"_path", false)
	sr.insertIndex(dc, ctrl, name, urlname+"_path", false)

	sr.insertOtherBase(dc, ctrl, urlname)
	sr.insertOtherItem(dc, ctrl, itemName)
	sr.insertWSBase(dc, ctrl, name, "ws_"+urlname+"_path", false)
	sr.insertWSItem(dc, ctrl, itemName, "ws_item_"+urlname+"_path", true)

	return &SubRoute{local: sr.local.InsertPath(itemName)}
}

func (sr *SubRoute) Namespace(name string) *SubRoute {
	return &SubRoute{local: sr.local.InsertPath(name)}
}

func (sr *SubRoute) Mount(m Module) {
	m.Load(sr)
}

func (sr *SubRoute) insertShow(dctrl DupableController, ctrl Controller, name, urlname string, item bool) {
	if _, ok := ctrl.(showController); ok {
		sr.local.Insert(
			name,
			Leaf{
				Method: "GET",
				Scheme: "http",
				Name:   urlname,
				Ctrl:   dctrl,
				Item:   item,
				Action: "Show",
				Callable: func(ctrl Controller) Result {
					if sc, ok := ctrl.(showController); ok {
						return sc.Show()
					}
					return InternalError{fmt.Errorf("BUG: controller passed is missing Show method")}
				},
			},
		)
	}
}

func (sr *SubRoute) insertEdit(dctrl DupableController, ctrl Controller, name, urlname string, item bool) {
	if _, ok := ctrl.(editController); ok {
		sr.local.Insert(
			name,
			Leaf{
				Method: "GET",
				Scheme: "http",
				Name:   urlname,
				Ctrl:   dctrl,
				Item:   item,
				Action: "Edit",
				Callable: func(ctrl Controller) Result {
					if uc, ok := ctrl.(editController); ok {
						return uc.Edit()
					}
					return InternalError{fmt.Errorf("BUG: controller passed is missing Edit method")}
				},
			},
		)
	}
}

func (sr *SubRoute) insertUpdate(dctrl DupableController, ctrl Controller, name, urlname string, item bool) {
	if _, ok := ctrl.(updateController); ok {
		sr.local.Insert(
			name,
			Leaf{
				Method: "POST",
				Scheme: "http",
				Name:   urlname,
				Ctrl:   dctrl,
				Item:   item,
				Action: "Update",
				Callable: func(ctrl Controller) Result {
					if uc, ok := ctrl.(updateController); ok {
						return uc.Update()
					}
					return InternalError{fmt.Errorf("BUG: controller passed is missing Update method")}
				},
			},
		)
	}
}

func (sr *SubRoute) insertNew(dctrl DupableController, ctrl Controller, name, urlname string, item bool) {
	if _, ok := ctrl.(newController); ok {
		sr.local.Insert(
			name,
			Leaf{
				Method: "GET",
				Scheme: "http",
				Name:   urlname,
				Ctrl:   dctrl,
				Item:   item,
				Action: "New",
				Callable: func(ctrl Controller) Result {
					if nc, ok := ctrl.(newController); ok {
						return nc.New()
					}
					return InternalError{fmt.Errorf("BUG: controller passed is missing New method")}
				},
			},
		)
	}
}

func (sr *SubRoute) insertCreate(dctrl DupableController, ctrl Controller, name, urlname string, item bool) {
	if _, ok := ctrl.(createController); ok {
		sr.local.Insert(
			name,
			Leaf{
				Method: "POST",
				Scheme: "http",
				Name:   urlname,
				Ctrl:   dctrl,
				Item:   item,
				Action: "Create",
				Callable: func(ctrl Controller) Result {
					if cc, ok := ctrl.(createController); ok {
						return cc.Create()
					}
					return InternalError{fmt.Errorf("BUG: controller passed is missing Create method")}
				},
			},
		)
	}
}

func (sr *SubRoute) insertDelete(dctrl DupableController, ctrl Controller, name, urlname string, item bool) {
	if _, ok := ctrl.(deleteController); ok {
		sr.local.Insert(
			name,
			Leaf{
				Method: "DELETE",
				Scheme: "http",
				Name:   urlname,
				Ctrl:   dctrl,
				Item:   item,
				Action: "Delete",
				Callable: func(ctrl Controller) Result {
					if dc, ok := ctrl.(deleteController); ok {
						return dc.Delete()
					}
					return InternalError{fmt.Errorf("BUG: controller passed is missing Delete method")}
				},
			},
		)
		sr.local.Insert(
			name+"/delete",
			Leaf{
				Method: "POST",
				Scheme: "http",
				Name:   urlname,
				Ctrl:   dctrl,
				Item:   item,
				Action: "Delete",
				Callable: func(ctrl Controller) Result {
					if dc, ok := ctrl.(deleteController); ok {
						return dc.Delete()
					}
					return InternalError{fmt.Errorf("BUG: controller passed is missing Delete method")}
				},
			},
		)
	}
}

func (sr *SubRoute) insertIndex(dctrl DupableController, ctrl Controller, name, urlname string, item bool) {
	if _, ok := ctrl.(indexController); ok {
		sr.local.Insert(
			name,
			Leaf{
				Method: "GET",
				Scheme: "http",
				Name:   urlname,
				Ctrl:   dctrl,
				Item:   item,
				Action: "Index",
				Callable: func(ctrl Controller) Result {
					if ic, ok := ctrl.(indexController); ok {
						return ic.Index()
					}
					return InternalError{fmt.Errorf("BUG: controller passed is missing Index method")}
				},
			},
		)
	}
}

func (sr *SubRoute) insertOtherBase(dctrl DupableController, ctrl Controller, name string) {
	if oc, ok := ctrl.(otherBaseController); ok {
		oc.OtherBase(&SubRoute{ctrl: dctrl, local: sr.local.InsertPath(name)})
	}
}
func (sr *SubRoute) insertOtherItem(dctrl DupableController, ctrl Controller, name string) {
	if oc, ok := ctrl.(otherItemController); ok {
		oc.OtherItem(&SubRoute{ctrl: dctrl, local: sr.local.InsertPath(name)})
	}
}

func (sr *SubRoute) insertWSBase(dctrl DupableController, ctrl Controller, name, urlname string, item bool) {
	if _, ok := ctrl.(wsBaseController); ok {
		sr.local.Insert(
			name,
			Leaf{
				Method: "GET",
				Scheme: "ws",
				Name:   urlname,
				Ctrl:   dctrl,
				Item:   item,
				Action: "WSBase",
				Callable: func(ctrl Controller) Result {
					if ic, ok := ctrl.(wsBaseController); ok {
						return &WSResult{
							Handler: websocket.Handler(ic.WSBase),
						}
					}
					return InternalError{fmt.Errorf("BUG: controller passed is missing WSBase method")}
				},
			},
		)
	}

}

func (sr *SubRoute) insertWSItem(dctrl DupableController, ctrl Controller, name, urlname string, item bool) {
	if _, ok := ctrl.(wsItemController); ok {
		sr.local.Insert(
			name,
			Leaf{
				Method: "GET",
				Scheme: "ws",
				Name:   urlname,
				Ctrl:   dctrl,
				Item:   item,
				Action: "WSItem",
				Callable: func(ctrl Controller) Result {
					if ic, ok := ctrl.(wsItemController); ok {
						return &UniqueHandler{
							Handler: websocket.Handler(ic.WSItem),
						}
					}

					return InternalError{fmt.Errorf("BUG: controller passed is missing Show method")}
				},
			},
		)
	}
}

func (sr *SubRoute) SetHandler(h http.Handler) {
	sr.local.Fallback = h
}
func (sr *SubRoute) Any(path string) Endpoint {
	return Endpoint{path, "*", sr}
}
func (sr *SubRoute) Get(path string) Endpoint {
	return Endpoint{path, "GET", sr}
}
func (sr *SubRoute) Post(path string) Endpoint {
	return Endpoint{path, "POST", sr}
}
func (sr *SubRoute) Put(path string) Endpoint {
	return Endpoint{path, "PUT", sr}
}
func (sr *SubRoute) Delete(path string) Endpoint {
	return Endpoint{path, "DELETE", sr}
}
func (sr *SubRoute) Other(verb, path string) Endpoint {
	return Endpoint{path, strings.ToUpper(verb), sr}
}
func (sr *SubRoute) WS(path string) WSEndpoint {
	return WSEndpoint{path, sr}
}

type Endpoint struct {
	path     string
	verb     string
	location *SubRoute
}

func (e Endpoint) HandlerFunc(f http.HandlerFunc) {
	e.location.local.Insert(
		e.path,
		Leaf{
			Method: e.verb,
			Scheme: "http",
			Ctrl:   &ctrlHF{handler: f},
			Item:   false,
			Action: "Custom",
			Callable: func(ctrl Controller) Result {
				if c, ok := ctrl.(*ctrlHF); ok {
					c.handler(c.w, c.r)
					return NothingResult{}
				}
				return NotFound{}
			},
		},
	)
}

func (e Endpoint) Action(a string) {
	e.location.local.Insert(
		e.path,
		Leaf{
			Method: e.verb,
			Scheme: "http",
			Ctrl:   e.location.ctrl,
			Item:   false,
			Action: a,
			Callable: func(ctrl Controller) Result {
				ac := reflect.ValueOf(ctrl).MethodByName(a)
				if ac.IsValid() {
					rr := ac.Call([]reflect.Value{})
					if len(rr) == 1 {
						if res, ok := rr[0].Interface().(Result); ok {
							return res
						}
					}
				}
				return NotFound{}
			},
		},
	)
}

type WSEndpoint struct {
	path     string
	location *SubRoute
}

type WSHandlerFunc func(c *websocket.Conn)

func (e WSEndpoint) WSHandlerFunc(f WSHandlerFunc) {
	e.location.local.Insert(
		e.path,
		Leaf{
			Scheme: "ws",
			Ctrl:   &ctrlHF{handler: wshandler(f)},
			Item:   false,
			Action: "Custom",
			Callable: func(ctrl Controller) Result {
				if c, ok := ctrl.(*ctrlHF); ok {
					c.handler(c.w, c.r)
					return NothingResult{}
				}
				return NotFound{}
			},
		},
	)
}

func (e WSEndpoint) Action(a string) {
	e.location.local.Insert(
		e.path,
		Leaf{
			Method: "GET",
			Scheme: "ws",
			Ctrl:   e.location.ctrl,
			Item:   false,
			Action: a,
			Callable: func(ctrl Controller) Result {
				ac := reflect.ValueOf(ctrl).MethodByName(a)
				if ac.IsValid() {
					if wh, ok := ac.Interface().(func(*websocket.Conn)); ok {
						return &WSResult{
							Handler: wh,
						}
					}
				}
				return NotFound{}
			},
		},
	)
}
