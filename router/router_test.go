package router

import "testing"

func TestRetrieveTree(t *testing.T) {
	rt := NewTree()
	rt.Insert("/users/*/edit", "users_edit_path")
	rt.Insert("/users/*", "users_show_path")
	rt.Insert("/users/*", "users_update_path")
	rt.Insert("/users/*", "users_delete_path")
	rt.Insert("/users", "users_index_path")
	rt.Insert("/users", "users_create_path")
	if len(rt.Retrieve("/users/123/edit")) != 1 {
		t.Fatal("Could not retrieve edit path")
	}

	if len(rt.Retrieve("/users/nerd")) != 3 {
		t.Fatal("Could not retrieve user path")
	}
	rt.Insert("/users/nerd", "users_nerd_category_path")
	if len(rt.Retrieve("/users/nerd")) != 1 {
		t.Fatal("Could not retrieve user path")
	}
	primary, secondary := rt.RetrieveWithFallback("/users/nerd")
	if len(primary) != 1 || len(secondary) != 3 {
		t.Fatal("Could not retrieve users path")
	}
	primary, secondary = rt.RetrieveWithFallback("/users/321/edit")
	if len(primary) != 1 || len(secondary) != 0 {
		t.Fatal("Could not retrieve users edit path")
	}
	primary, secondary = rt.RetrieveWithFallback("/users/nerd/edit")
	if len(primary) != 1 || len(secondary) != 0 {
		t.Fatal("Could not retrieve users edit for special path", primary, secondary)
	}

	if len(rt.Retrieve("/users/")) != 2 {
		t.Fatal("Could not retrieve users index path")
	}

}
