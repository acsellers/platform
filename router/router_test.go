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
}

func (t t1Ctrl) Show() error {
	return nil
}

func (t t1Ctrl) Path() string {
	return "foo"
}

func (t t1Ctrl) Dupe() Controller {
	return t1Ctrl{&BaseController{}}
}

type t2Ctrl struct {
	*BaseController
}

func (t t2Ctrl) New() error {
	return nil
}
func (t t2Ctrl) Create() error {
	return nil
}

func (t t2Ctrl) Edit() error {
	return nil
}

func (t t2Ctrl) Path() string {
	return "bar"
}

func (t t2Ctrl) Dupe() Controller {
	return t2Ctrl{&BaseController{}}
}

func TestRouter(t *testing.T) {
	r := NewRouter()
	r.One(t1Ctrl{})
	r.Many(t2Ctrl{})
	rl := r.RouteList()
	if len(rl) != 4 {
		fmt.Println(r.RouteList())
		t.Fatal("RouteList not correct")
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
}
