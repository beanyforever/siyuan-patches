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

//go:build fts5

package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/88250/lute/parse"
	"github.com/siyuan-note/siyuan/kernel/cache"
	"github.com/siyuan-note/siyuan/kernel/conf"
	"github.com/siyuan-note/siyuan/kernel/filesys"
	"github.com/siyuan-note/siyuan/kernel/sql"
	"github.com/siyuan-note/siyuan/kernel/treenode"
	"github.com/siyuan-note/siyuan/kernel/util"
)

// moveCacheFixture is the file operation fixture's notebook, Work, with SQL's databases and a second notebook,
// Personal, and every cache emptied, as after a restart.
type moveCacheFixture struct {
	work, personal string
}

func setupMoveCacheTest(t *testing.T) *moveCacheFixture {
	t.Helper()
	files := setupStructureTransactionTest(t)
	setupFoldTransactionDatabase(t, files)
	personal := &Box{ID: "20261016100000-persnal"}
	boxConf := conf.NewBoxConf()
	boxConf.Name = "Personal"
	boxConf.Closed = false
	if err := personal.SaveConf(boxConf); nil != err {
		t.Fatalf("save the second notebook's conf failed: %v", err)
	}
	clear := func() {
		cache.ClearTreeCache()
		cache.ClearBlocksIAL()
		cache.ClearDocsIAL()
	}
	clear()
	t.Cleanup(clear)
	return &moveCacheFixture{work: files.box.ID, personal: personal.ID}
}

func addMoveCachePage(t *testing.T, boxID, p, hpath, title string) *parse.Tree {
	t.Helper()
	tree := treenode.NewTree(boxID, p, hpath, title)
	if _, err := filesys.WriteTree(tree); nil != err {
		t.Fatalf("write page %s failed: %v", p, err)
	}
	treenode.UpsertBlockTree(tree)
	return tree
}

// appendMoveCacheParagraph is /api/block/appendBlock: an edit, through the transaction an editor's edit takes.
func appendMoveCacheParagraph(t *testing.T, parentID, id, text string) {
	t.Helper()
	dom := util.NewLute().Md2BlockDOM(text+"\n{: id=\""+id+"\"}", false)
	if _, err := PerformBlockOperation(&Operation{Action: "appendInsert", ParentID: parentID, Data: dom}); nil != err {
		t.Fatalf("append %s to %s failed: %v", text, parentID, err)
	}
}

// setMoveCacheAttr is /api/attr/setBlockAttrs.
func setMoveCacheAttr(t *testing.T, id, name, value string) {
	t.Helper()
	if err := SetBlockAttrs(id, map[string]string{name: value}); nil != err {
		t.Fatalf("set %s=%s on %s failed: %v", name, value, id, err)
	}
}

// moveMoveCachePage is /api/filetree/moveDocsByID to a notebook's top level.
func moveMoveCachePage(t *testing.T, id, toBoxID string) {
	t.Helper()
	bt := treenode.GetBlockTree(id)
	if nil == bt {
		t.Fatalf("page %s isn't in the block index", id)
	}
	if err := MoveDocs([]string{bt.Path}, toBoxID, "/", nil); nil != err {
		t.Fatalf("move %s to %s failed: %v", id, toBoxID, err)
	}
}

// moveCachePageFile reads a page as stored: its file's text and its attributes.
func moveCachePageFile(t *testing.T, id string) (text string, attrs map[string]string) {
	t.Helper()
	bt := treenode.GetBlockTree(id)
	if nil == bt {
		t.Fatalf("page %s isn't in the block index", id)
	}
	absPath := filepath.Join(util.DataDir, bt.BoxID, bt.Path)
	data, err := os.ReadFile(absPath)
	if nil != err {
		t.Fatalf("read %s failed: %v", absPath, err)
	}
	if attrs, err = filesys.DocIAL(absPath); nil != err {
		t.Fatalf("read the attributes of %s failed: %v", absPath, err)
	}
	return string(data), attrs
}

// A page moved with its sub-page to another notebook, edited and given an Attribute there, then moved back and edited
// again, keeps everything it was given, as polka's triage from Work to Personal and back does to a page. On SiYuan
// v3.8.7 the move leaves the page cache's entries and the cached Attributes under the old notebook, so back home the
// page and its sub-page load as they were when they left: getBlockAttrs serves the Attributes from before, the block
// index, rebuilt from those loads, loses the blocks added while away, and the next edit saves the old copy over the
// file, losing the edit made away.
func TestMoveToAnotherNotebookAndBackKeepsEdits(t *testing.T) {
	f := setupMoveCacheTest(t)
	const pageID, subID = "20261016100001-pagexxx", "20261016100002-subpage"
	addMoveCachePage(t, f.work, "/"+pageID+".sy", "/Page", "Page")
	addMoveCachePage(t, f.work, "/"+pageID+"/"+subID+".sy", "/Page/Sub", "Sub")

	// In Work: an edit to each page, then an Attribute on the page.
	appendMoveCacheParagraph(t, pageID, "20261016100011-editina", "EDIT-IN-A")
	appendMoveCacheParagraph(t, subID, "20261016100021-subedta", "SUB-EDIT-IN-A")
	setMoveCacheAttr(t, pageID, "custom-set-in-a", "1")

	moveMoveCachePage(t, pageID, f.personal)

	// In Personal: an Attribute on the page, then an edit to each page.
	setMoveCacheAttr(t, pageID, "custom-set-in-b", "1")
	appendMoveCacheParagraph(t, pageID, "20261016100012-editinb", "EDIT-IN-B")
	appendMoveCacheParagraph(t, subID, "20261016100022-subedtb", "SUB-EDIT-IN-B")

	moveMoveCachePage(t, pageID, f.work)

	addedAway := map[string]string{"20261016100012-editinb": pageID, "20261016100022-subedtb": subID}
	checkReads := func(step string) {
		t.Helper()
		_, fileAttrs := moveCachePageFile(t, pageID)
		attrs := sql.GetBlockAttrs(pageID)
		for _, name := range []string{"custom-set-in-a", "custom-set-in-b"} {
			if fileAttrs[name] != attrs[name] {
				t.Errorf("%s: getBlockAttrs gives %s=%q, the page's file %q", step, name, attrs[name], fileAttrs[name])
			}
		}
		for blockID, rootID := range addedAway {
			if bt := treenode.GetBlockTree(blockID); nil == bt || f.work != bt.BoxID || rootID != bt.RootID {
				t.Errorf("%s: the block index doesn't find block %s, added to %s while it was away, in Work: %+v",
					step, blockID, rootID, bt)
			}
		}
		for _, id := range []string{pageID, subID} {
			bt := treenode.GetBlockTree(id)
			if nil == bt {
				t.Errorf("%s: the block index doesn't find page %s", step, id)
				continue
			}
			text, _ := moveCachePageFile(t, id)
			loaded, err := filesys.LoadTree(bt.BoxID, bt.Path, util.NewLute())
			if nil != err {
				t.Fatalf("%s: load %s failed: %v", step, id, err)
			}
			for blockID, rootID := range addedAway {
				if rootID == id && strings.Contains(text, blockID) && nil == treenode.GetNodeInTree(loaded, blockID) {
					t.Errorf("%s: page %s loads without block %s, which its file holds", step, id, blockID)
				}
			}
		}
	}
	checkReads("back in Work")

	// Back in Work: one more edit to each page.
	appendMoveCacheParagraph(t, pageID, "20261016100013-editret", "EDIT-AFTER-RETURN")
	appendMoveCacheParagraph(t, subID, "20261016100023-subedtr", "SUB-EDIT-AFTER-RETURN")

	pageText, pageAttrs := moveCachePageFile(t, pageID)
	for _, edit := range []string{"EDIT-IN-A", "EDIT-IN-B", "EDIT-AFTER-RETURN"} {
		if !strings.Contains(pageText, edit) {
			t.Errorf("after an edit back in Work, the page's file lacks %s", edit)
		}
	}
	for _, name := range []string{"custom-set-in-a", "custom-set-in-b"} {
		if "1" != pageAttrs[name] {
			t.Errorf("after an edit back in Work, the page's file lacks the Attribute %s", name)
		}
	}
	subText, _ := moveCachePageFile(t, subID)
	for _, edit := range []string{"SUB-EDIT-IN-A", "SUB-EDIT-IN-B", "SUB-EDIT-AFTER-RETURN"} {
		if !strings.Contains(subText, edit) {
			t.Errorf("after an edit back in Work, the sub-page's file lacks %s", edit)
		}
	}
	checkReads("after an edit back in Work")
}

// A page edited, then given a new Type, then moved to another notebook, reads as its new Type, as polka's change_type
// on an edited Note does (audit #10, Case B). On SiYuan v3.8.7 the edit caches the page's Attributes under the bare
// key, the new Type goes under the notebook's key, and after the move getBlockAttrs finds nothing under the new
// notebook's key and falls back to the edit's copy, which still says note.
func TestMoveAfterEditReadsNewType(t *testing.T) {
	f := setupMoveCacheTest(t)
	const pageID = "20261016100003-shedpln"
	addMoveCachePage(t, f.work, "/"+pageID+".sy", "/Shed plans", "Shed plans")
	setMoveCacheAttr(t, pageID, "custom-object-type", "note")
	appendMoveCacheParagraph(t, pageID, "20261016100031-typofix", "fix a typo")
	setMoveCacheAttr(t, pageID, "custom-object-type", "task")
	moveMoveCachePage(t, pageID, f.personal)

	if _, fileAttrs := moveCachePageFile(t, pageID); "task" != fileAttrs["custom-object-type"] {
		t.Fatalf("setup: the page's file says custom-object-type=%q", fileAttrs["custom-object-type"])
	}
	if got := sql.GetBlockAttrs(pageID)["custom-object-type"]; "task" != got {
		t.Fatalf("after the move, getBlockAttrs gives custom-object-type=%q, though the page's file says task", got)
	}
}
