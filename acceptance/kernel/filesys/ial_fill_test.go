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

package filesys

import (
	"testing"

	"github.com/88250/lute/parse"
	"github.com/siyuan-note/siyuan/kernel/cache"
	"github.com/siyuan-note/siyuan/kernel/treenode"
	"github.com/siyuan-note/siyuan/kernel/util"
)

// Writing a page's file refuses the block attribute fills whose readers took their token before it: they may have
// read the page from before the write, and a page write changes its blocks' attributes without writing them to the
// block attribute cache. WriteTreeIfUnchanged, which find and replace and asset relinking write with, does the same.
func TestWriteTreeRefusesBlockFillsTakenBefore(t *testing.T) {
	cases := []struct {
		name  string
		write func(t *testing.T, tree *parse.Tree)
	}{
		{"WriteTree", func(t *testing.T, tree *parse.Tree) {
			tree.Root.SetIALAttr("tags", "written") // a write of the same bytes is skipped, and changes nothing
			if _, err := WriteTree(tree); err != nil {
				t.Fatal(err)
			}
		}},
		{"WriteTreeIfUnchanged", func(t *testing.T, tree *parse.Tree) {
			loaded, original, err := LoadTreeSnapshot(tree.Box, tree.Path, util.NewLute())
			if err != nil {
				t.Fatal(err)
			}
			loaded.Root.SetIALAttr("tags", "replaced")
			if _, err = WriteTreeIfUnchanged(loaded, original); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			box, root, _, _ := setupStalePathTest(t)
			t.Cleanup(cache.ClearBlocksIAL)
			tree := treenode.NewTree(box, "/"+root+".sy", "/Page", "Page")
			if _, err := WriteTree(tree); err != nil {
				t.Fatal(err)
			}
			attrs := parse.IAL2Map(tree.Root.KramdownIAL)

			token := cache.NewFillToken()
			c.write(t, tree)
			if cache.FillBlockIALInBox(tree.ID, tree.ID, box, attrs, token) {
				t.Fatal("a block fill whose token came before a write of its page stored")
			}
			if !cache.FillBlockIALInBox(tree.ID, tree.ID, box, attrs, cache.NewFillToken()) {
				t.Fatal("a block fill whose token came after the page's write didn't store")
			}
		})
	}
}
