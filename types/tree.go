package types

import (
	"bytes"
	"strings"

	"github.com/a8m/djson"
	"github.com/summadb/summadb/utils"
)

type Tree struct {
	Leaf
	Branches
	Rev     string
	Map     string
	Reduce  string
	Deleted bool
	Key     string

	// fields for requesting values on Select()
	RequestLeaf    bool
	RequestRev     bool
	RequestMap     bool
	RequestReduce  bool
	RequestDeleted bool
	RequestKey     bool
}

type Branches map[string]*Tree

func NewTree() *Tree { return &Tree{Branches: make(Branches)} }

func (t *Tree) UnmarshalJSON(j []byte) error {
	v, err := djson.Decode(j)
	if err != nil {
		return err
	}

	*t = TreeFromInterface(v)
	return nil
}

func TreeFromInterface(v interface{}) Tree {
	t := Tree{}

	switch val := v.(type) {
	case map[interface{}]interface{}:
		newmap := make(map[string]interface{}, len(val))
		for keyname, value := range val {
			newmap[keyname.(string)] = value
		}
		return TreeFromInterface(newmap)
	case map[string]interface{}:
		if key, ok := val["_key"]; ok {
			t.Key = key.(string)
		}
		if val, ok := val["_val"]; ok {
			t.Leaf = LeafFromInterface(val)
		}
		if rev, ok := val["_rev"]; ok {
			t.Rev = rev.(string)
		}
		if mapf, ok := val["!map"]; ok {
			t.Map = mapf.(string)
		}
		if reducef, ok := val["!reduce"]; ok {
			t.Reduce = reducef.(string)
		}
		if deleted, ok := val["_del"]; ok {
			t.Deleted = deleted.(bool)
		}

		delete(val, "_key")
		delete(val, "_val")
		delete(val, "_rev")
		delete(val, "!map")
		delete(val, "!reduce")
		delete(val, "_del")
		t.Branches = make(Branches, len(val))
		for k, v := range val {
			subt := TreeFromInterface(v)
			t.Branches[k] = &subt
		}
	default:
		t.Leaf = LeafFromInterface(v)
	}
	return t
}

func (t Tree) MarshalJSON() ([]byte, error) {
	var parts [][]byte

	// current leaf
	if t.Leaf.Kind != UNDEFINED {
		jsonLeaf, err := t.Leaf.MarshalJSON()
		if err != nil {
			return nil, err
		}
		buffer := bytes.NewBufferString(`"_val":`)
		buffer.Write(jsonLeaf)
		parts = append(parts, buffer.Bytes())
	}

	// key
	if t.Key != "" {
		buffer := bytes.NewBufferString(`"_key":`)
		buffer.Write(utils.JSONString(t.Key))
		parts = append(parts, buffer.Bytes())
	}

	// rev
	if t.Rev != "" {
		buffer := bytes.NewBufferString(`"_rev":`)
		buffer.Write(utils.JSONString(t.Rev))
		parts = append(parts, buffer.Bytes())
	}

	// map
	if t.Map != "" {
		buffer := bytes.NewBufferString(`"!map":`)
		buffer.WriteString(`"` + strings.Replace(t.Map, `"`, `\"`, -1) + `"`)
		parts = append(parts, buffer.Bytes())
	}

	// reduce
	if t.Reduce != "" {
		buffer := bytes.NewBufferString(`"!reduce":`)
		buffer.WriteString(`"` + strings.Replace(t.Reduce, `"`, `\"`, -1) + `"`)
		parts = append(parts, buffer.Bytes())
	}

	// deleted
	if t.Deleted {
		buffer := bytes.NewBufferString(`"_del":`)
		buffer.WriteString("true")
		parts = append(parts, buffer.Bytes())
	}

	// all branches
	if len(t.Branches) > 0 {
		subts := make([][]byte, len(t.Branches))
		i := 0
		for k, Tree := range t.Branches {
			jsonLeaf, err := Tree.MarshalJSON()
			if err != nil {
				return nil, err
			}
			subts[i] = append([]byte(`"`+strings.Replace(k, `"`, `\"`, -1)+`":`), jsonLeaf...)
			i++
		}
		joinedbranches := bytes.Join(subts, []byte{','})
		parts = append(parts, joinedbranches)
	}

	joined := bytes.Join(parts, []byte{','})
	out := append([]byte{'{'}, joined...)
	out = append(out, '}')
	return out, nil
}

func (t Tree) Recurse(p Path, handle func(Path, Leaf, Tree) bool) {
	proceed := handle(p, t.Leaf, t)
	if proceed {
		for key, t := range t.Branches {
			deeppath := p.Child(key)
			t.Recurse(deeppath, handle)
		}
	}
}

func (t *Tree) DeepPath(p Path) *Tree {
	var exists bool
	for _, k := range p {
		if t, exists = t.Branches[k]; !exists {
			t = NewTree()
		}
	}
	return t
}

func (t Tree) ToInterface() map[string]interface{} {
	o := map[string]interface{}{}

	// current leaf
	if t.Leaf.Kind != NULL {
		o["_val"] = t.Leaf.ToInterface()
	}

	// key
	if t.Key != "" {
		o["_key"] = t.Key
	}

	// rev
	if t.Rev != "" {
		o["_rev"] = t.Rev
	}

	// map
	if t.Map != "" {
		o["!map"] = t.Map
	}

	// deleted
	if t.Deleted {
		o["_del"] = t.Deleted
	}

	// all branches
	for subkey, branch := range t.Branches {
		o[subkey] = branch.ToInterface()
	}

	return o
}
