// SiYuan - From thought to insight, with agents
// Copyright (c) 2020-present, b3log.org
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package sql

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/88250/lute/parse"
	"github.com/siyuan-note/siyuan/kernel/cache"
	"github.com/siyuan-note/siyuan/kernel/filesys"
	"github.com/siyuan-note/siyuan/kernel/treenode"
	"github.com/siyuan-note/siyuan/kernel/util"
)

const fillTestBox = "20261016090000-notebk1"

// A write straight after a read that filled the cache keeps the write. "+ New" in a filtered database view renders the
// view to place the new row, which reads the new page's attributes from its file (BatchGetBlockAttrs,
// getBlockAttrsFromTree) and caches them without custom-avs; a fraction of a millisecond later the row's binding
// writes them with custom-avs. Without the fix ristretto turns that write away, as a second put of a key it doesn't
// hold yet, and the page reads as bound to no database.
func TestBlockAttrsFillThenWriteKeepsWrite(t *testing.T) {
	cache.ClearBlocksIAL()
	t.Cleanup(cache.ClearBlocksIAL)
	const avID = "20261016090000-databse"
	lost := 0
	for i := range 100 {
		p := fmt.Sprintf("/20261016090%03d-fillnew.sy", i)
		tree := treenode.NewTree(fillTestBox, p, "/New row", "New row")
		token := cache.NewFillToken()
		if read := getBlockAttrsFromTree(tree.ID, tree, &token); "" == read["id"] {
			t.Fatalf("setup: the read found no attributes for %s: %v", tree.ID, read)
		}
		bound := parse.IAL2Map(tree.Root.KramdownIAL)
		bound["custom-avs"] = avID
		cache.PutBlockIALInBox(tree.ID, fillTestBox, bound) // the binding's write, as setNodeAttrsWithTx makes it
		if avID != cache.GetBlockIALWithBoxFallback(tree.ID, fillTestBox)["custom-avs"] {
			lost++
		}
	}
	if 0 < lost {
		t.Fatalf("a write straight after a fill was lost: %d of 100 pages read without custom-avs", lost)
	}
}

// A read that took no token before its first lookup caches nothing it reads. The kanban and gallery renders hand
// BatchGetBlockAttrsWitTrees trees they loaded earlier, which may be older than a write the cache has since lost.
func TestBatchGetBlockAttrsWitTreesCachesNothing(t *testing.T) {
	cache.ClearBlocksIAL()
	t.Cleanup(cache.ClearBlocksIAL)
	tree := treenode.NewTree(fillTestBox, "/20261016090000-witrees.sy", "/Card", "Card")
	read := BatchGetBlockAttrsWitTrees([]string{tree.ID}, map[string]*parse.Tree{tree.ID: tree})
	if "" == read[tree.ID]["id"] {
		t.Fatalf("setup: the read found no attributes: %v", read)
	}
	if got := cache.GetBlockIALWithBoxFallback(tree.ID, fillTestBox); nil != got {
		t.Fatalf("a read of a tree its caller loaded earlier cached what it read: %v", got)
	}
}

// The reads that load a page take their token before their first lookup, and so cache what they load: the single
// read, and the batch read, which a database render makes, "+ New" among them. A read that took no token would cache
// nothing.
func TestBlockAttrsReadsFillWhatTheyLoad(t *testing.T) {
	originalDataDir, originalBlockTreeDBPath := util.DataDir, util.BlockTreeDBPath
	util.DataDir = t.TempDir()
	util.BlockTreeDBPath = filepath.Join(t.TempDir(), "blocktree.db")
	treenode.InitBlockTree(true)
	t.Cleanup(func() {
		cache.ClearBlocksIAL()
		cache.ClearTreeCache()
		treenode.CloseDatabase()
		util.DataDir, util.BlockTreeDBPath = originalDataDir, originalBlockTreeDBPath
	})
	tree := treenode.NewTree(fillTestBox, "/20261016090000-loaded1.sy", "/Loaded", "Loaded")
	if _, err := filesys.WriteTree(tree); err != nil {
		t.Fatal(err)
	}
	treenode.UpsertBlockTree(tree)

	for _, c := range []struct {
		name string
		read func() map[string]string
	}{
		{"GetBlockAttrs", func() map[string]string { return GetBlockAttrs(tree.ID) }},
		{"BatchGetBlockAttrs", func() map[string]string { return BatchGetBlockAttrs([]string{tree.ID})[tree.ID] }},
	} {
		t.Run(c.name, func(t *testing.T) {
			cache.ClearBlocksIAL()
			if got := c.read(); tree.ID != got["id"] {
				t.Fatalf("setup: the read gave %v", got)
			}
			if got := cache.GetBlockIALInBox(tree.ID, fillTestBox); tree.ID != got["id"] {
				t.Fatalf("the read didn't cache what it loaded: the cache holds %v", got)
			}
		})
	}
}
