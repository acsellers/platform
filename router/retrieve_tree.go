package router

import "strings"

type RetrieveTree struct {
	*Branch
}

type Branch struct {
	Static  map[string]*Branch
	Name    string
	Path    string
	Dynamic *Branch
	Leaves  []Leaf
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

func (rt RetrieveTree) Retrieve(path string) ([]Leaf, map[string]string) {
	if rt.Branch == nil {
		return []Leaf{}, nil
	}

	splits := strings.Split(path, "/")
	current := rt.Branch
	params := map[string]string{}
	for _, split := range splits {
		if split == "" {
			continue
		}
		if current.Static == nil && current.Dynamic == nil {
			return []Leaf{}, nil
		}

		if br, ok := current.Static[split]; ok {
			current = br
		} else if current.Dynamic != nil {
			current = current.Dynamic
			params[current.Name] = split
		} else {
			return []Leaf{}, nil
		}
	}
	return current.Leaves, params
}

func (rt RetrieveTree) RetrieveWithFallback(path string) ([]Leaf, []Leaf, map[string]string) {
	params := map[string]string{}
	if rt.Branch == nil {
		return []Leaf{}, []Leaf{}, params
	}

	splits := strings.Split(path, "/")
	current := rt.Branch
	backtrack := current.Dynamic

	for _, split := range splits {
		if split == "" {
			continue
		}
		if current.Static == nil && current.Dynamic == nil && backtrack == nil {
			return []Leaf{}, []Leaf{}, params
		}

		if br, ok := current.Static[split]; ok {
			current = br
			if current.Dynamic != nil {
				backtrack = current.Dynamic
				params["backtrack_value"] = split
			}
		} else if current.Dynamic != nil {
			current = current.Dynamic
			params[current.Name] = split
			backtrack = nil
			delete(params, "backtrack_value")

		} else if backtrack != nil {
			params[backtrack.Name] = params["backtrack_value"]
			delete(params, "backtrack_value")

			if br, ok := backtrack.Static[split]; ok {
				current = br
				if backtrack.Dynamic != nil {
					backtrack = backtrack.Dynamic
					params["backtrack_value"] = split
				} else {
					backtrack = nil
				}
			} else if backtrack.Dynamic != nil {
				current = backtrack.Dynamic
				backtrack = nil
				delete(params, "backtrack_value")
			} else {
				return []Leaf{}, []Leaf{}, params
			}
		} else {
			return []Leaf{}, []Leaf{}, params
		}
	}
	if backtrack != nil {
		return current.Leaves, backtrack.Leaves, params
	} else {
		return current.Leaves, []Leaf{}, params
	}
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
	dc := item.Ctrl.Dupe()
	_, scok := dc.(contexter)
	item.SetContext = scok
	_, pfok := dc.(prefilter)
	item.PreFilter = pfok
	_, piok := dc.(preitem)
	item.PreItem = piok

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
	Name                 string
	Item                 bool
	Action               string
	Path                 string
	Ctrl                 DupableController
	Callable             func(Controller) Result
	SetContext, SetCache bool
	PreFilter, PreItem   bool
}
