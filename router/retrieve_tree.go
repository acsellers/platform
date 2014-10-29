package router

import "strings"

type RetrieveTree struct {
	*Branch
}

type Branch struct {
	Static  map[string]*Branch
	Dynamic *Branch
	Leaves  []string
}

func NewTree() *RetrieveTree {
	return &RetrieveTree{&Branch{}}
}
func (rt *RetrieveTree) Insert(path, item string) *Branch {
	if rt.Branch == nil {
		rt.Branch = &Branch{}
	}

	return rt.Branch.Insert(path, item)
}

func (rt RetrieveTree) Retrieve(path string) []string {
	if rt.Branch == nil {
		return []string{}
	}

	splits := strings.Split(path, "/")
	current := rt.Branch
	for _, split := range splits {
		if split == "" {
			continue
		}
		if current.Static == nil && current.Dynamic == nil {
			return []string{}
		}

		if br, ok := current.Static[split]; ok {
			current = br
		} else if current.Dynamic != nil {
			current = current.Dynamic
		} else {
			return []string{}
		}
	}
	return current.Leaves
}

func (rt RetrieveTree) RetrieveWithFallback(path string) ([]string, []string) {
	if rt.Branch == nil {
		return []string{}, []string{}
	}

	splits := strings.Split(path, "/")
	current := rt.Branch
	var backtrack *Branch
	for _, split := range splits {
		if split == "" {
			continue
		}
		if current.Static == nil && current.Dynamic == nil && backtrack == nil {
			return []string{}, []string{}
		}

		if br, ok := current.Static[split]; ok {
			current = br
			if current.Dynamic != nil {
				backtrack = current.Dynamic
			}
		} else if current.Dynamic != nil {
			current = current.Dynamic
			backtrack = nil
		} else if backtrack != nil {
			if br, ok := backtrack.Static[split]; ok {
				current = br
				if backtrack.Dynamic != nil {
					backtrack = backtrack.Dynamic
				} else {
					backtrack = nil
				}
			} else if backtrack.Dynamic != nil {
				current = backtrack.Dynamic
				backtrack = nil
			}
		} else {
			return []string{}, []string{}
		}
	}
	if backtrack != nil {
		return current.Leaves, backtrack.Leaves
	} else {
		return current.Leaves, []string{}
	}
}

func (b *Branch) Insert(path, item string) *Branch {
	splits := strings.Split(path, "/")
	current := b
	for _, split := range splits {
		if split == "" {
			continue
		}
		switch {
		// Dynamic Option
		case split == "*" && current.Dynamic == nil:
			current.Dynamic = &Branch{}
			fallthrough
		case split == "*":
			current = current.Dynamic

		// Static Option
		case current.Static == nil:
			current.Static = make(map[string]*Branch)
			fallthrough
		case current.Static[split] == nil:
			current.Static[split] = &Branch{}
			fallthrough
		default:
			current = current.Static[split]
		}
	}
	current.Leaves = append(current.Leaves, item)

	return current
}
