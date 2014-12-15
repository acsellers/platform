package router

import (
	"fmt"
	"testing"
)

func TestRetrieveTree(t *testing.T) {
	rt := NewTree()
	rt.Insert("/users/:usersid/edit", Leaf{Name: "users_edit_path"})
	rt.Insert("/users/:usersid", Leaf{Name: "users_show_path"})
	rt.Insert("/users/:usersid", Leaf{Name: "users_update_path"})
	rt.Insert("/users/:usersid", Leaf{Name: "users_delete_path"})
	rt.Insert("/users", Leaf{Name: "users_index_path"})
	rt.Insert("/users", Leaf{Name: "users_create_path"})
	results, _ := rt.Retrieve("/users/123/edit")
	if len(results) != 1 {
		t.Fatal("Could not retrieve edit path")
	}

	results, _ = rt.Retrieve("/users/nerd")
	if len(results) != 3 {
		t.Fatal("Could not retrieve user path")
	}
	rt.Insert("/users/nerd", Leaf{Name: "users_nerd_category_path"})
	results, _ = rt.Retrieve("/users/nerd")
	if len(results) != 1 {
		t.Fatal("Could not retrieve user path")
	}
	primary, secondary, _ := rt.RetrieveWithFallback("/users/nerd")
	if len(primary) != 1 || len(secondary) != 3 {
		t.Fatal("Could not retrieve users path")
	}
	primary, secondary, _ = rt.RetrieveWithFallback("/users/321/edit")
	if len(primary) != 1 || len(secondary) != 0 {
		t.Fatal("Could not retrieve users edit path")
	}
	primary, secondary, _ = rt.RetrieveWithFallback("/users/nerd/edit")
	if len(primary) != 1 || len(secondary) != 0 {
		t.Fatal("Could not retrieve users edit for special path", primary, secondary)
	}
	results, _ = rt.Retrieve("/users/")
	if len(results) != 2 {
		t.Fatal("Could not retrieve users index path")
	}
	results, _ = rt.Retrieve("/users")
	if len(results) != 2 {
		t.Fatal("Could not retrieve users index path")
	}
}

type t1Ctrl struct {
	*BaseController
	Custom string
}

func (t t1Ctrl) Show() Result {
	return nil
}

func (t t1Ctrl) Get(attr string) string {
	return fmt.Sprint(t.Context[attr])
}

func (t t1Ctrl) Path() string {
	if t.Custom != "" {
		return t.Custom
	}
	return "foo"
}

type t3Ctrl struct {
	t1Ctrl
}

type t2Ctrl struct {
	*BaseController
}

func (t t2Ctrl) New() Result {
	return nil
}
func (t t2Ctrl) Create() Result {
	return nil
}

func (t t2Ctrl) Edit() Result {
	return nil
}

func (t t2Ctrl) Index() Result {
	return nil
}

func (t t2Ctrl) Show() Result {
	return nil
}

func (t t2Ctrl) Path() string {
	return "bar"
}

func TestRouter(t *testing.T) {
	r := NewRouter()
	r.One(t1Ctrl{})
	t2 := r.Many(t2Ctrl{})
	t2.Many(t1Ctrl{})
	rl := r.RouteList()
	if len(rl) != 7 {
		t.Fatal("RouteList not correct:", r.RouteList())
	}
	results, _ := r.Tree.Retrieve("/bar/new")
	if len(results) != 1 {
		fmt.Println("Coundn't find new bar handler", results)
		t.Fatal("RouteList doesn't have bar new")
	}

	results, _ = r.Tree.Retrieve("/bar/123/edit")
	if len(results) != 1 {
		fmt.Println("Coundn't find edit bar handler", results)
		t.Fatal("RouteList doesn't have bar edit")
	}
	results, _, _ = r.Tree.RetrieveWithFallback("/bar")
	if len(results) != 2 {
		fmt.Println("Coundn't find index + create bar handler", results)
		t.Fatal("RouteList doesn't have bar index + create")
	}

	results, _ = r.Tree.Retrieve("/foo")
	if len(results) != 1 {
		fmt.Println("Coundn't find show foohandler", results)
		t.Fatal("RouteList doesn't have foo show")

	}

	results, _ = r.Tree.Retrieve("/bar/123/foo/afafa")
	if len(results) != 1 {
		fmt.Println("Coundn't find show foohandler", results)
		t.Fatal("RouteList doesn't have foo show")
	}
}
func TestDupingBasic(t *testing.T) {
	tc := t1Ctrl{&BaseController{}, "asdf"}
	dt := autoDupeCtrl{tc}

	d1 := dt.Dupe()
	d2 := dt.Dupe()
	if d1.Path() != d2.Path() || d1.Path() != tc.Path() {
		t.Fatal("Bad things")
	}
	ctx := map[string]interface{}{
		"andrew": "sellers",
	}
	if cc, ok := d1.(interface {
		SetContext(map[string]interface{})
	}); ok {
		cc.SetContext(ctx)
	}

	ctx2 := map[string]interface{}{
		"andrew": "blah",
	}
	if cc, ok := d2.(interface {
		SetContext(map[string]interface{})
	}); ok {
		cc.SetContext(ctx2)
	}

	if gc, ok := d1.(interface {
		Get(string) string
	}); ok {
		if gc.Get("andrew") != "sellers" {
			t.Fatal("incorrect context")
		}
	}

	if gc, ok := d2.(interface {
		Get(string) string
	}); ok {
		if gc.Get("andrew") == "sellers" {
			t.Fatal("incorrect context")
		}
	}

}

func TestDupingNested(t *testing.T) {
	tc := t3Ctrl{t1Ctrl{&BaseController{}, "asdf"}}
	dt := autoDupeCtrl{tc}

	d1 := dt.Dupe()
	d2 := dt.Dupe()
	if d1.Path() != d2.Path() || d1.Path() != tc.Path() {
		t.Fatal("Bad things")
	}
	ctx := map[string]interface{}{
		"andrew": "sellers",
	}
	if cc, ok := d1.(interface {
		SetContext(map[string]interface{})
	}); ok {
		cc.SetContext(ctx)
	}

	ctx2 := map[string]interface{}{
		"andrew": "blah",
	}
	if cc, ok := d2.(interface {
		SetContext(map[string]interface{})
	}); ok {
		cc.SetContext(ctx2)
	}

	if gc, ok := d1.(interface {
		Get(string) string
	}); ok {
		if gc.Get("andrew") != "sellers" {
			t.Fatal("incorrect context")
		}
	}

	if gc, ok := d2.(interface {
		Get(string) string
	}); ok {
		if gc.Get("andrew") == "sellers" {
			t.Fatal("incorrect context")
		}
	}

}
