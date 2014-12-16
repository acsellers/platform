package router

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type Router struct {
	Tree      *RetrieveTree
	BadRoute  http.HandlerFunc
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

	handlers, fallbacks, params := r.Tree.RetrieveWithFallback(req.URL.Path)
	reqLog.Printf("%d Possible Handlers, %d Fallback Handlers", len(handlers), len(fallbacks))
	if len(handlers) == 0 && len(fallbacks) == 0 {
		reqLog.Println("Bad Route", req.URL.Path)
		// r.BadRoute(w, req)
		return
	}

	for _, handler := range append(handlers, fallbacks...) {
		if handler.Method != req.Method {
			reqLog.Printf("Skipping %s.%s due to incorrect method\n", ctrlName(handler.Ctrl), handler.Action)
			continue
		}
		// prepare
		ctrl, res := callCtrl(w, req, handler, params, reqLog)
		if res != nil {
			if _, ok := res.(NotFound); ok {
				reqLog.Println("Aborting current handler, starting next handler")
				continue
			}
			res.Execute(w)
			reqLog.Println(res)
			return
		}
		res = handler.Callable(ctrl)
		if res == nil {
			return
		}
		res.Execute(w)
		reqLog.Println(res)
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

func (r *Router) Namespace(name string) *SubRoute {
	sr := SubRoute{local: r.Tree.Branch}
	return sr.Namespace(name)
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

	return &SubRoute{local: sr.local.InsertPath(name)}
}

func (sr *SubRoute) Many(ctrl Controller) *SubRoute {
	var dc DupableController
	var ok bool
	if dc, ok = ctrl.(DupableController); !ok {
		dc = autoDupeCtrl{ctrl}
	}
	name := ctrl.Path()
	itemName := fmt.Sprintf("%[1]s/:%[1]sid", name)
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

	return &SubRoute{local: sr.local.InsertPath(itemName)}
}

func (sr *SubRoute) Namespace(name string) *SubRoute {
	return &SubRoute{local: sr.local.InsertPath(name)}
}

func (sr *SubRoute) insertShow(dctrl DupableController, ctrl Controller, name, urlname string, item bool) {
	if sc, ok := ctrl.(showController); ok {
		insert := true
		if rc, ok := ctrl.(beenReset); ok {
			insert = !rc.resetFunc("Show", fmt.Sprint(sc.Show))
		}
		if insert {
			sr.local.Insert(
				name,
				Leaf{
					Method: "GET",
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
}

func (sr *SubRoute) insertEdit(dctrl DupableController, ctrl Controller, name, urlname string, item bool) {
	if uc, ok := ctrl.(editController); ok {
		insert := true
		if rc, ok := ctrl.(beenReset); ok {
			insert = !rc.resetFunc("Edit", fmt.Sprint(uc.Edit))
		}
		if insert {
			sr.local.Insert(
				name,
				Leaf{
					Method: "GET",
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
}

func (sr *SubRoute) insertUpdate(dctrl DupableController, ctrl Controller, name, urlname string, item bool) {
	if uc, ok := ctrl.(updateController); ok {
		insert := true
		if rc, ok := ctrl.(beenReset); ok {
			insert = !rc.resetFunc("Update", fmt.Sprint(uc.Update))
		}
		if insert {
			sr.local.Insert(
				name,
				Leaf{
					Method: "POST",
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
}

func (sr *SubRoute) insertNew(dctrl DupableController, ctrl Controller, name, urlname string, item bool) {
	if nc, ok := ctrl.(newController); ok {
		insert := true
		if rc, ok := ctrl.(beenReset); ok {
			insert = !rc.resetFunc("New", fmt.Sprint(nc.New))
		}
		if insert {
			sr.local.Insert(
				name,
				Leaf{
					Method: "GET",
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
}

func (sr *SubRoute) insertCreate(dctrl DupableController, ctrl Controller, name, urlname string, item bool) {
	if cc, ok := ctrl.(createController); ok {
		insert := true
		if rc, ok := ctrl.(beenReset); ok {
			insert = !rc.resetFunc("Create", fmt.Sprint(cc.Create))
		}
		if insert {
			sr.local.Insert(
				name,
				Leaf{
					Method: "POST",
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
}

func (sr *SubRoute) insertDelete(dctrl DupableController, ctrl Controller, name, urlname string, item bool) {
	if dc, ok := ctrl.(deleteController); ok {
		insert := true
		if rc, ok := ctrl.(beenReset); ok {
			insert = !rc.resetFunc("Delete", fmt.Sprint(dc.Delete))
		}
		if insert {
			sr.local.Insert(
				name,
				Leaf{
					Method: "DELETE",
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
}

func (sr *SubRoute) insertIndex(dctrl DupableController, ctrl Controller, name, urlname string, item bool) {
	if ic, ok := ctrl.(indexController); ok {
		insert := true
		if rc, ok := ctrl.(beenReset); ok {
			insert = !rc.resetFunc("Index", fmt.Sprint(ic.Index))
		}
		if insert {
			sr.local.Insert(
				name,
				Leaf{
					Method: "GET",
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
}
