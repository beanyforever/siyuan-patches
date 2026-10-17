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
	"time"

	"github.com/88250/lute/parse"
	"github.com/siyuan-note/siyuan/kernel/cache"
	"github.com/siyuan-note/siyuan/kernel/treenode"
	"github.com/siyuan-note/siyuan/kernel/util"
)

// Both of the page cache's fills cache nothing they read while a write of the page's file is under way, as a save is
// between starting to write the file and caching its bytes: a LoadTree that missed the cache, and a WriteTree that
// finds the file already holds its bytes, so skips writing and caches what it read. Once no write is under way, each
// caches the file as before. Unlike TestTreeFillRacesWrite, this places the read inside the write, so it shows each
// fill's check on its own, WriteTree's included, whose read and store are too close together to race from a test.
func TestTreeFillsDuringFileWriteCacheNothing(t *testing.T) {
	cases := []struct {
		name string
		read func(t *testing.T, tree *parse.Tree)
	}{
		{"LoadTree", func(t *testing.T, tree *parse.Tree) {
			if _, err := LoadTree(tree.Box, tree.Path, util.NewLute()); err != nil {
				t.Fatal(err)
			}
		}},
		{"WriteTree of the bytes the file holds", func(t *testing.T, tree *parse.Tree) {
			if size, err := WriteTree(tree); err != nil || 0 != size {
				t.Fatalf("a WriteTree of the bytes the file holds wrote %d bytes: %v", size, err)
			}
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			box, root, _, _ := setupStalePathTest(t)
			tree := treenode.NewTree(box, "/"+root+".sy", "/Page", "Page")
			if _, err := WriteTree(tree); err != nil {
				t.Fatal(err)
			}
			cache.RemoveTreeDataInBox(tree.ID, box) // as after a restart

			cache.BeginTreeWrite(tree.ID) // another save of the page is writing its file
			c.read(t, tree)
			time.Sleep(5 * time.Millisecond) // let ristretto take queued sets in
			_, cached := cache.GetTreeDataInBox(tree.ID, box)
			cache.EndTreeWrite(tree.ID)
			if cached {
				t.Fatal("a read cached the page's file while a write of it was under way")
			}

			c.read(t, tree)
			deadline := time.Now().Add(time.Second)
			for {
				if _, ok := cache.GetTreeDataInBox(tree.ID, box); ok {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("a read nothing raced didn't cache the page's file")
				}
				time.Sleep(time.Millisecond)
			}
		})
	}
}
