package router

import (
	"net/http"
	"strings"
)

type RetrieveTree struct {
	*Branch
}

type Branch struct {
	Static   map[string]*Branch
	Name     string
	Path     string
	Dynamic  *Branch
	Fallback http.Handler
	Leaves   []Leaf
}

func NewTree() *RetrieveTree {
	return &RetrieveTree{&Branch{}}
}

func (rt *RetrieveTree) Insert(path string, item Leaf) *Branch {
	if rt.Branch == nil {
		rt.Branch = &Branch{}
	}

	return rt.Branch.Insert(path, item)
}

type Matches struct {
	Primary   []Leaf
	Secondary []Leaf
	Fallback  http.Handler
	Params    map[string]string
}

func (rt RetrieveTree) Retrieve(path string) Matches {
	r := Matches{
		Params: map[string]string{},
	}
	if rt.Branch == nil {
		return r
	}

	splits := strings.Split(path, "/")
	current := rt.Branch
	for _, split := range splits {
		if split == "" {
			continue
		}

		if current.Fallback != nil {
			r.Fallback = current.Fallback
		}
		if current.Static == nil && current.Dynamic == nil {
			return r
		}

		if br, ok := current.Static[split]; ok {
			current = br
		} else if current.Dynamic != nil {
			current = current.Dynamic
			r.Params[current.Name] = split
		} else {
			return r
		}
	}
	r.Primary = current.Leaves
	return r
}

func (rt RetrieveTree) RetrieveWithFallback(path string) Matches {
	r := Matches{
		Params: map[string]string{},
	}
	if rt.Branch == nil {
		return r
	}

	splits := strings.Split(path, "/")
	current := rt.Branch
	backtrack := current.Dynamic

	for _, split := range splits {
		if split == "" {
			continue
		}
		if current.Static == nil && current.Dynamic == nil && backtrack == nil {
			return r
		}

		if current.Fallback != nil {
			r.Fallback = current.Fallback
		}
		if br, ok := current.Static[split]; ok {
			current = br
			if current.Dynamic != nil {
				backtrack = current.Dynamic
				r.Params["backtrack_value"] = split
			}
		} else if current.Dynamic != nil {
			current = current.Dynamic
			r.Params[current.Name] = split
			backtrack = nil
			delete(r.Params, "backtrack_value")

		} else if backtrack != nil {
			r.Params[backtrack.Name] = r.Params["backtrack_value"]
			delete(r.Params, "backtrack_value")

			if br, ok := backtrack.Static[split]; ok {
				current = br
				if backtrack.Dynamic != nil {
					backtrack = backtrack.Dynamic
					r.Params["backtrack_value"] = split
				} else {
					backtrack = nil
				}
			} else if backtrack.Dynamic != nil {
				current = backtrack.Dynamic
				backtrack = nil
				delete(r.Params, "backtrack_value")
			} else {
				return r
			}
		} else {
			return r
		}
	}
	r.Primary = current.Leaves
	if backtrack != nil {
		r.Secondary = backtrack.Leaves
	}
	return r
}
func (b *Branch) InsertPath(path string) *Branch {
	splits := strings.Split(path, "/")
	current := b
	if path == "" {
		return b
	}
	for _, split := range splits {
		if split == "" {
			continue
		}
		switch {
		// Dynamic Option
		case split[0] == ':' && current.Dynamic == nil:
			current.Dynamic = &Branch{Name: split, Path: current.Path + "/" + split}
			fallthrough
		case split[0] == ':':
			current = current.Dynamic

		// Static Option
		case current.Static == nil:
			current.Static = make(map[string]*Branch)
			fallthrough
		case current.Static[split] == nil:
			current.Static[split] = &Branch{Path: current.Path + "/" + split}
			fallthrough
		default:
			current = current.Static[split]
		}
	}
	return current
}

type contexter interface {
	SetContext(map[string]interface{})
}

type prefilter interface {
	PreFilter() Result
}

type preitem interface {
	PreItem() Result
}

func (b *Branch) Insert(path string, item Leaf) *Branch {
	br := b.InsertPath(path)
	if item.Ctrl != nil {
		dc := item.Ctrl.Dupe()
		_, scok := dc.(contexter)
		item.SetContext = scok
		_, pfok := dc.(prefilter)
		item.PreFilter = pfok
		_, piok := dc.(preitem)
		item.PreItem = piok
	}

	item.Path = br.Path
	br.Leaves = append(br.Leaves, item)
	return br
}

func (rt RetrieveTree) ListLeaves() []Leaf {
	br := []*Branch{rt.Branch}
	lf := []Leaf{}
	for len(br) > 0 {
		brz := br[0]
		br = br[1:]
		lf = append(lf, brz.Leaves...)
		for _, branch := range brz.Static {
			br = append(br, branch)
		}
		if brz.Dynamic != nil {
			br = append(br, brz.Dynamic)
		}
	}
	return lf
}

type Leaf struct {
	Method               string
	Scheme               string
	Name                 string
	Item                 bool
	Action               string
	Path                 string
	Ctrl                 DupableController
	Callable             func(Controller) Result
	SetContext, SetCache bool
	PreFilter, PreItem   bool
}
