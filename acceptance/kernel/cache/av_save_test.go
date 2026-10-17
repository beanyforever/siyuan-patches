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
	"bytes"
	"fmt"
	"testing"
)

// A read straight after a database save gets the save from the cache. Each save stores its copy under a fresh
// ristretto key and drops the live one at once, and ristretto takes the new key in later, so without the fix a read in
// between misses, reads the file and stores what it read: the moment in which a read that loaded the file before a
// save can store the older copy as the newest. The first save of a database, and every save after it, are checked, and
// so is the normalized copy the first parse after a save stores the same way.
func TestAVCacheReadAfterSaveHits(t *testing.T) {
	const avID, boxID = "20261016090000-avsaved", "20261016090000-notebk1"
	t.Run("save", func(t *testing.T) {
		ClearAVCache()
		t.Cleanup(ClearAVCache)
		misses := 0
		for i := range 100 {
			raw := []byte(fmt.Sprintf(`{"name":"save %d"}`, i))
			version := SetAVDataWithVersionInBox(avID, boxID, raw)
			got, gotVersion, ok := GetAVDataWithVersionInBox(avID, boxID)
			if !ok {
				misses++
				continue
			}
			if !bytes.Equal(got, raw) || gotVersion != version {
				t.Fatalf("save %d: a read straight after it got %q (version %d), not the save (version %d)", i, got, gotVersion, version)
			}
		}
		if 0 < misses {
			t.Fatalf("a read straight after a save missed the cache, and would read the file, after %d of 100 saves", misses)
		}
	})
	t.Run("normalized copy", func(t *testing.T) {
		ClearAVCache()
		t.Cleanup(ClearAVCache)
		misses := 0
		for i := range 100 {
			raw := []byte(fmt.Sprintf(`{"name":"save %d"}`, i))
			version := SetAVDataWithVersionInBox(avID, boxID, raw)
			avCache.Wait() // the save itself is in, whatever the fix to it
			normalized := []byte(fmt.Sprintf(`{"name":"save %d","normalized":true}`, i))
			if !SetAVNormalizedDataInBox(avID, boxID, version, raw, normalized) {
				t.Fatalf("save %d: the normalized copy wasn't stored", i)
			}
			got, ok := GetAVNormalizedDataInBox(avID, boxID, version)
			_, _, rawOK := GetAVDataWithVersionInBox(avID, boxID)
			if !ok || !rawOK {
				misses++
				continue
			}
			if !bytes.Equal(got, normalized) {
				t.Fatalf("save %d: a read straight after storing the normalized copy got %q", i, got)
			}
		}
		if 0 < misses {
			t.Fatalf("a read straight after storing a normalized copy missed the cache after %d of 100 saves", misses)
		}
	})
}
