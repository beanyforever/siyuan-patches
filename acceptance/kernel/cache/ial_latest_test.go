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

package cache

import (
	"fmt"
	"testing"
)

// Two puts of a block's or a document's attributes back to back, the second before the cache has taken in the first,
// as making a row with "+ New" does: once both have returned, a read gives the second. ristretto queues a put of a key
// it doesn't hold yet, and then turns away a second put of that key as one it already holds, so without waiting the
// read gives the first put's attributes, here a page's from before it was bound to its row.
func TestIALCacheKeepsLatestPut(t *testing.T) {
	cases := []struct {
		name  string
		put   func(id, boxID string, ial map[string]string)
		get   func(id, boxID string) map[string]string
		wait  func()
		clear func()
	}{
		{"block", PutBlockIALInBox, GetBlockIALInBox, blockIALCache.Wait, ClearBlocksIAL},
		{"document", PutDocIALInBox, GetDocIALInBox, docIALCache.Wait, ClearDocsIAL},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Cleanup(c.clear)
			for i := range 100 {
				id := fmt.Sprintf("20261016090000-latest%d", i)
				c.put(id, "box", map[string]string{"id": id})
				c.put(id, "box", map[string]string{"id": id, "custom-avs": "20261016090000-database"})
				c.wait() // whatever the cache still has queued, it has now taken in
				if got := c.get(id, "box"); got["custom-avs"] != "20261016090000-database" {
					t.Fatalf("after two puts of %s, a read gave %v, not the second put's attributes", id, got)
				}
			}
		})
	}
}
