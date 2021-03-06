package database

import (
	"errors"

	slu "github.com/fiatjaf/levelup/stringlevelup"
	"github.com/summadb/summadb/types"
)

type QueryParams struct {
	KeyStart   string
	KeyEnd     string
	Descending bool
	Limit      int
}

// Query provide a querying interface similar to CouchDB, in which you can manually specify
// key start and end, starting at a certain "path level".
// in contrast with Read, which returns a big tree of everything under the given path,
// Query return an array of trees, as the children of the given path.
func (db *SummaDB) Query(sourcepath types.Path, params QueryParams) (records []*types.Tree, err error) {
	if !sourcepath.ReadValid() {
		return records, errors.New("cannot read invalid path: " + sourcepath.Join())
	}

	rangeopts := slu.RangeOpts{
		Start:   sourcepath.Child("").Join(),
		End:     sourcepath.Child("~~~").Join(),
		Reverse: params.Descending,
	}
	if params.KeyStart != "" {
		rangeopts.Start = sourcepath.Join() + "/" + params.KeyStart
	}
	if params.KeyEnd != "" {
		rangeopts.End = sourcepath.Join() + "/" + params.KeyEnd
	}

	if params.Limit == 0 {
		// default to a large number
		params.Limit = 99999
	}

	// once we find something is deleted we'll remove it, but it will try to get readded by its children rows
	deleted := make(map[string]bool)

	iter := db.ReadRange(&rangeopts)
	defer iter.Release()
	for ; iter.Valid(); iter.Next() {
		if err = iter.Error(); err != nil {
			return
		}

		path := types.ParsePath(iter.Key())
		relpath := path.RelativeTo(sourcepath)

		// the first key of the relpath is the _key
		key := relpath[0]
		relpath = relpath[1:]

		// special keys of the sourcepath shouldn't count as records
		if key[0] == '_' || key[0] == '!' {
			continue
		}

		// skip records already found to be deleted
		if _, is := deleted[key]; is {
			continue
		}

		// fetch the tree we're currently filling or start a new tree
		var tree *types.Tree
		if len(records) == 0 || records[len(records)-1].Key != key {
			// will start a new tree, if allowed by our 'limit' clause
			if params.Limit == len(records) {
				return
			}
			tree = types.NewTree()
			tree.Key = key
			records = append(records, tree)
		} else {
			// fetched the tree we're currently filling
			tree = records[len(records)-1]
		}

		value := iter.Value()
		if value == "" {
			continue
		}

		// descend into tree filling in the values read from the database
		currentbranch := tree

		for i := 0; i <= len(relpath); i++ {
			if i == len(relpath) {
				// we're past the last key, so we're finished. add the leaf here.
				leaf := &types.Leaf{}
				if err = leaf.UnmarshalJSON([]byte(value)); err != nil {
					log.Error("failed to unmarshal json leaf on Query()",
						"value", value,
						"err", err)
					return
				}
				currentbranch.Leaf = *leaf
			} else {
				key := relpath[i]

				// special values should be added as special values, not branches
				switch key {
				case "_rev":
					currentbranch.Rev = value
				case "!map":
					if i == len(relpath)-1 {
						// grab the code for the map function, never any of its results
						currentbranch.Map = value
					}
				case "!reduce":
					if i == len(relpath)-1 {
						// grab the code for the reduce function, never any of its results
						currentbranch.Reduce = value
					}
				case "_del":
					currentbranch.Deleted = true
					if i == 0 {
						// deleted records are not be fetched, so we'll remove this from the results
						records = records[:len(records)-1]
						deleted[currentbranch.Key] = true
					}
				default:
					// create a subbranch at this key
					subbranch, exists := currentbranch.Branches[key]
					if !exists {
						subbranch = types.NewTree()
						currentbranch.Branches[key] = subbranch
					}

					// proceed to the next, deeper, branch
					currentbranch = subbranch
					continue
				}
				break // will break if it is a special key, continue if not
			}
		}
	}
	return
}
