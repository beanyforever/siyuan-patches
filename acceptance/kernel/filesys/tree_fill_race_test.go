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
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/88250/lute/ast"
	"github.com/siyuan-note/siyuan/kernel/cache"
	"github.com/siyuan-note/siyuan/kernel/treenode"
	"github.com/siyuan-note/siyuan/kernel/util"
)

// A page cache fill racing a page write keeps the write. A LoadTree that misses the page cache reads the file and
// then caches what it read; a WriteTree on another goroutine writes the file and caches its own bytes. Nothing orders
// the two stores, so without the fix the fill can land last and leave the page cache older than the file, and the
// next edit, which loads the page from the cache, writes that older copy back over the file. A page isn't cached after
// a restart or a move, and a read of it while an edit to it is saved is ordinary. Rounds are many, with jitter, since
// only the timing differs; with a FillToken taken before the fill's lookup, none go stale.
func TestTreeFillRacesWrite(t *testing.T) {
	box, _, _, _ := setupStalePathTest(t)
	p := "/20261016000000-racefil.sy"
	tree := treenode.NewTree(box, p, "/Race", "Race")
	tree.Root.SetIALAttr("custom-n", "0")
	// A page of realistic size, so that parsing it takes as long as it does in use.
	for j := range 200 {
		paragraph := treenode.NewParagraph("")
		paragraph.AppendChild(&ast.Node{Type: ast.NodeText, Tokens: []byte("Some paragraph text that makes the page a realistic size " + strconv.Itoa(j))})
		tree.Root.AppendChild(paragraph)
	}
	if _, err := WriteTree(tree); err != nil {
		t.Fatal(err)
	}
	filePath := filepath.Join(util.DataDir, box, p)
	const rounds = 1000
	stale, lost := 0, 0
	for i := 1; i <= rounds; i++ {
		cache.RemoveTreeDataInBox(tree.ID, box) // the next load misses
		want := strconv.Itoa(i)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); _, _ = LoadTree(box, p, util.NewLute()) }()
		jitter := time.Duration(rand.Intn(400)) * time.Microsecond
		go func() {
			defer wg.Done()
			time.Sleep(jitter)
			tree.Root.SetIALAttr("custom-n", want)
			if _, err := WriteTree(tree); err != nil {
				t.Error(err)
			}
		}()
		wg.Wait()
		time.Sleep(5 * time.Millisecond) // let ristretto take queued sets in

		disk, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatal(err)
		}
		onDisk, err := LoadTreeByData(disk, box, p, util.NewLute())
		if err != nil {
			t.Fatal(err)
		}
		if got := onDisk.Root.IALAttr("custom-n"); got != want {
			t.Fatalf("round %d: the file holds custom-n=%q, want %q", i, got, want)
		}
		// The next edit loads the page as SiYuan does, from the cache first, and saves it.
		loaded, err := LoadTree(box, p, util.NewLute())
		if err != nil {
			t.Fatal(err)
		}
		if got := loaded.Root.IALAttr("custom-n"); got != want {
			stale++
			loaded.Root.SetIALAttr("custom-other", want)
			if _, err = WriteTree(loaded); err != nil {
				t.Fatal(err)
			}
			disk, _ = os.ReadFile(filePath)
			if after, _ := LoadTreeByData(disk, box, p, util.NewLute()); nil != after && after.Root.IALAttr("custom-n") != want {
				lost++
			}
			// Put the file and the cache back in step for the next round.
			if _, err = WriteTree(tree); err != nil {
				t.Fatal(err)
			}
		}
	}
	t.Logf("%d rounds: the page cache was older than the file after %d, and the next edit lost the write from the file after %d", rounds, stale, lost)
	if 0 < stale {
		t.Fatalf("a page cache fill racing a page write left the cache older than the file in %d of %d rounds, "+
			"and the next edit lost the write from the file in %d", stale, rounds, lost)
	}
}
