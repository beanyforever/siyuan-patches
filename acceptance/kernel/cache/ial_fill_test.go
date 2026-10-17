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

// A fill is a read that missed the cache, loaded the file and caches what it read. It takes a FillToken before its
// first lookup and before it loads, and lands only if nothing wrote or removed the key since.

const fillTestBox = "20261016090000-notebk1"

func olderIAL(id string) map[string]string {
	return map[string]string{"id": id, "custom-object-type": "note"}
}

// A block fill also checks the page it read the block from: a page write bumps no block's version, so a fill that
// loaded the page before the write, or while it was being written, could otherwise store the block's older
// attributes.
func TestIALBlockFillChecksItsPage(t *testing.T) {
	const id, root = "20261016090000-fillblk", "20261016090000-fillpge"
	fill := func(token FillToken) bool { return FillBlockIALInBox(id, root, fillTestBox, olderIAL(id), token) }
	t.Cleanup(ClearBlocksIAL)

	ClearBlocksIAL()
	token := NewFillToken()
	BeginTreeWrite(root)
	EndTreeWrite(root)
	if fill(token) {
		t.Fatal("a block fill whose token came before a write of its page stored")
	}

	BeginTreeWrite(root)
	if fill(NewFillToken()) {
		t.Fatal("a block fill stored while its page was being written")
	}
	EndTreeWrite(root)

	token = NewFillToken()
	RemoveTreeData(root)
	if fill(token) {
		t.Fatal("a block fill whose page was removed from the page cache after its token stored")
	}

	if !fill(NewFillToken()) {
		t.Fatal("a block fill that took its token after its page's last write didn't store")
	}
}

// fillVersions forgets every key once it holds too many, and refuses the fills that began before, so memory stays
// bounded at the cost of a few refused fills.
func TestFillVersionsForgetPastTheirLimit(t *testing.T) {
	v := newFillVersions()
	token := NewFillToken()
	for i := range fillVersionsLimit + 1 {
		v.bump(fmt.Sprintf("key%d", i))
	}
	if fillVersionsLimit < len(v.last) {
		t.Fatalf("fillVersions holds %d keys, past its limit of %d", len(v.last), fillVersionsLimit)
	}
	if v.fresh("never-written", token) {
		t.Fatal("a fill that began before fillVersions forgot its keys was let through")
	}
	if !v.fresh("never-written", NewFillToken()) {
		t.Fatal("a fill that began after fillVersions forgot its keys was refused")
	}
}
