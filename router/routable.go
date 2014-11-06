package router

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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
		ctrl, err := callCtrl(w, req, handler, params, reqLog)
		if err != nil {
			if _, ok := err.(NotApplicable); ok {
				reqLog.Println("Aborting current handler, starting next handler")
				continue
			}
			if re, ok := err.(RedirectError); ok {
				reqLog.Println(err)
				re.Redirect(w, req)
				return
			}
			if r.OnError != nil {
				reqLog.Printf("Encountered error: %s\n", err.Error())
				r.OnError(err, w, req, ctrl)
			} else {
				reqLog.Printf("Encountered unknown error (%s) with no OnError handler\n", err.Error())
				return
			}
		}
		err = handler.Callable(ctrl)
		if err == nil {
			return
		}
		if re, ok := err.(RedirectError); ok {
			reqLog.Println(err)
			re.Redirect(w, req)
			return
		}
		if _, ok := err.(NotApplicable); ok {
			reqLog.Println("Aborting current handler, starting next handler")
			continue
		}
		if r.OnError != nil {
			reqLog.Printf("Encountered error: %s\n", err.Error())
			r.OnError(err, w, req, ctrl)
			return
		} else {
			reqLog.Printf("Encountered unknown error (%s) with no OnError handler\n", err.Error())
			return
		}
	}
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
	Show() error
}
type editController interface {
	Edit() error
}
type updateController interface {
	Update() error
}
type deleteController interface {
	Delete() error
}
type memberController interface {
	Member() map[string]Member
}

type newController interface {
	New() error
}
type createController interface {
	Create() error
}
type indexController interface {
	Index() error
}
type collectionController interface {
	Collection() map[string]Collection
}

func (sr *SubRoute) One(ctrl Controller) *SubRoute {
	name := ctrl.Path()
	urlname := name
	if len(sr.name) > 0 {
		urlname = name + "_" + sr.name
	}

	sr.insertShow(ctrl, name, urlname+"_path", false)
	sr.insertEdit(ctrl, name+"/edit", "edit_"+urlname+"_path", false)
	sr.insertUpdate(ctrl, name, "update_"+urlname+"_path", false)
	sr.insertDelete(ctrl, name, "delete_"+urlname+"_path", false)

	return &SubRoute{local: sr.local.InsertPath(name)}
}

func (sr *SubRoute) Many(ctrl Controller) *SubRoute {
	name := ctrl.Path()
	itemName := fmt.Sprintf("%[1]s/:%[1]sid", name)
	urlname := name
	if len(sr.name) > 0 {
		urlname = name + "_" + sr.name
	}

	sr.insertShow(ctrl, itemName, "show_"+urlname+"_path", true)
	sr.insertEdit(ctrl, itemName+"/edit", "edit_"+urlname+"_path", true)
	sr.insertUpdate(ctrl, itemName, "update_"+urlname+"_path", true)
	sr.insertDelete(ctrl, itemName, "delete_"+urlname+"_path", true)

	sr.insertNew(ctrl, name+"/new", "new_"+urlname+"_path", false)
	sr.insertCreate(ctrl, name, "create_"+urlname+"_path", false)
	sr.insertIndex(ctrl, name, urlname+"_path", false)

	return &SubRoute{local: sr.local.InsertPath(itemName)}
}

func (sr *SubRoute) Namespace(name string) *SubRoute {
	if _, ok := sr.local.Static[name]; !ok {
		sr.local.Static[name] = &Branch{}
	}
	return &SubRoute{local: sr.local.Static[name]}
}

func (sr *SubRoute) insertShow(ctrl Controller, name, urlname string, item bool) {
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
					Ctrl:   ctrl,
					Item:   item,
					Action: "Show",
					Callable: func(ctrl Controller) error {
						if sc, ok := ctrl.(showController); ok {
							return sc.Show()
						}
						return fmt.Errorf("BUG: controller passed is missing Show method")
					},
				},
			)
		}
	}
}

func (sr *SubRoute) insertEdit(ctrl Controller, name, urlname string, item bool) {
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
					Ctrl:   ctrl,
					Item:   item,
					Action: "Edit",
					Callable: func(ctrl Controller) error {
						if uc, ok := ctrl.(editController); ok {
							return uc.Edit()
						}
						return fmt.Errorf("BUG: controller passed is missing Edit method")
					},
				},
			)
		}
	}
}

func (sr *SubRoute) insertUpdate(ctrl Controller, name, urlname string, item bool) {
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
					Ctrl:   ctrl,
					Item:   item,
					Action: "Update",
					Callable: func(ctrl Controller) error {
						if uc, ok := ctrl.(updateController); ok {
							return uc.Update()
						}
						return fmt.Errorf("BUG: controller passed is missing Update method")
					},
				},
			)
		}
	}
}

func (sr *SubRoute) insertNew(ctrl Controller, name, urlname string, item bool) {
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
					Ctrl:   ctrl,
					Item:   item,
					Action: "New",
					Callable: func(ctrl Controller) error {
						if nc, ok := ctrl.(newController); ok {
							return nc.New()
						}
						return fmt.Errorf("BUG: controller passed is missing New method")
					},
				},
			)
		}
	}
}

func (sr *SubRoute) insertCreate(ctrl Controller, name, urlname string, item bool) {
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
					Ctrl:   ctrl,
					Item:   item,
					Action: "Create",
					Callable: func(ctrl Controller) error {
						if cc, ok := ctrl.(createController); ok {
							return cc.Create()
						}
						return fmt.Errorf("BUG: controller passed is missing Create method")
					},
				},
			)
		}
	}
}

func (sr *SubRoute) insertDelete(ctrl Controller, name, urlname string, item bool) {
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
					Ctrl:   ctrl,
					Item:   item,
					Action: "Delete",
					Callable: func(ctrl Controller) error {
						if dc, ok := ctrl.(deleteController); ok {
							return dc.Delete()
						}
						return fmt.Errorf("BUG: controller passed is missing Delete method")
					},
				},
			)
			sr.local.Insert(
				name+"/delete",
				Leaf{
					Method: "POST",
					Name:   urlname,
					Ctrl:   ctrl,
					Item:   item,
					Action: "Delete",
					Callable: func(ctrl Controller) error {
						if dc, ok := ctrl.(deleteController); ok {
							return dc.Delete()
						}
						return fmt.Errorf("BUG: controller passed is missing Delete method")
					},
				},
			)
		}
	}
}

func (sr *SubRoute) insertIndex(ctrl Controller, name, urlname string, item bool) {
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
					Ctrl:   ctrl,
					Item:   item,
					Action: "Index",
					Callable: func(ctrl Controller) error {
						if ic, ok := ctrl.(indexController); ok {
							return ic.Index()
						}
						return fmt.Errorf("BUG: controller passed is missing Index method")
					},
				},
			)
		}
	}
}
