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
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/siyuan-note/siyuan/kernel/cache"
	"github.com/siyuan-note/siyuan/kernel/conf"
	"github.com/siyuan-note/siyuan/kernel/filesys"
	"github.com/siyuan-note/siyuan/kernel/sql"
	"github.com/siyuan-note/siyuan/kernel/treenode"
	"github.com/siyuan-note/siyuan/kernel/util"
)

const (
	restoreCachePageID      = "20261016110000-restore"
	restoreCacheParagraphID = "20261016110000-shelf01" // on the page in the version restored, and deleted since
	restoreCacheQuoteID     = "20261016110000-quote01" // a blockquote added after the version restored
	restoreCacheQuotedID    = "20261016110000-quoted1" // the paragraph in it
)

// setupRestoreCacheTest is the file operation fixture with SQL's databases, a workspace folder holding the history
// folder as SiYuan lays it out, every cache emptied, and one page in the fixture's notebook whose Type is note and
// whose paragraph's custom-shelf is left.
func setupRestoreCacheTest(t *testing.T) (boxID, pagePath string) {
	t.Helper()
	files := setupStructureTransactionTest(t)
	setupFoldTransactionDatabase(t, files)
	originalWorkspaceDir, originalHistoryDir := util.WorkspaceDir, util.HistoryDir
	util.WorkspaceDir = t.TempDir()
	util.HistoryDir = filepath.Join(util.WorkspaceDir, "history")
	clear := func() {
		cache.ClearTreeCache()
		cache.ClearBlocksIAL()
		cache.ClearDocsIAL()
	}
	clear()
	t.Cleanup(func() {
		// The rollbacks refresh the page on a goroutine of their own once SQL's queue is flushed.
		sql.FlushQueue()
		time.Sleep(200 * time.Millisecond)
		clear()
		util.WorkspaceDir, util.HistoryDir = originalWorkspaceDir, originalHistoryDir
	})
	boxID, pagePath = files.box.ID, "/"+restoreCachePageID+".sy"
	tree := treenode.NewTree(boxID, pagePath, "/Shed plans", "Shed plans")
	tree.Root.AppendChild(treenode.NewParagraph(restoreCacheParagraphID))
	if _, err := filesys.WriteTree(tree); nil != err {
		t.Fatal(err)
	}
	treenode.UpsertBlockTree(tree)
	setRestoreCacheAttr(t, restoreCachePageID, "custom-object-type", "note")
	setRestoreCacheAttr(t, restoreCacheParagraphID, "custom-shelf", "left")
	return
}

// setRestoreCacheAttr is /api/attr/setBlockAttrs, setting one of a block's attributes, checked through getBlockAttrs.
func setRestoreCacheAttr(t *testing.T, id, name, value string) {
	t.Helper()
	if err := SetBlockAttrs(id, map[string]string{name: value}); nil != err {
		t.Fatalf("set %s=%s on %s failed: %v", name, value, id, err)
	}
	if got := sql.GetBlockAttrs(id)[name]; value != got {
		t.Fatalf("setup: after setting %s=%s on %s, getBlockAttrs gives %q", name, value, id, got)
	}
}

// changeRestoreCachePage changes the page after the version a test restores: its Type becomes task, its paragraph's
// custom-shelf right before the paragraph is deleted, and it gains a blockquote whose custom-shelf is quote. Deleting
// a block leaves its attributes cached, and editing the paragraph in the blockquote caches the blockquote's attributes
// under its bare id, as every edit does for the blocks around the one it edits.
func changeRestoreCachePage(t *testing.T, boxID string) {
	t.Helper()
	setRestoreCacheAttr(t, restoreCachePageID, "custom-object-type", "task")
	setRestoreCacheAttr(t, restoreCacheParagraphID, "custom-shelf", "right")
	if err := PerformTransactionSync(&Transaction{DoOperations: []*Operation{{Action: "delete", ID: restoreCacheParagraphID}}}); nil != err {
		t.Fatalf("delete the paragraph failed: %v", err)
	}
	if "right" != cache.GetBlockIALWithBoxFallback(restoreCacheParagraphID, boxID)["custom-shelf"] {
		t.Fatal("setup: deleting the paragraph dropped its cached attributes")
	}
	quote := util.NewLute().Md2BlockDOM(fmt.Sprintf("> a trowel\n> {: id=\"%s\"}\n{: id=\"%s\"}", restoreCacheQuotedID, restoreCacheQuoteID), false)
	if err := PerformTransactionSync(&Transaction{DoOperations: []*Operation{{Action: "appendInsert", ParentID: restoreCachePageID, Data: quote}}}); nil != err {
		t.Fatalf("add the blockquote failed: %v", err)
	}
	setRestoreCacheAttr(t, restoreCacheQuoteID, "custom-shelf", "quote")
	edit := fmt.Sprintf(`<div data-node-id="%s" data-type="NodeParagraph"><div contenteditable="true">a trowel</div></div>`, restoreCacheQuotedID)
	if err := PerformTransactionSync(&Transaction{DoOperations: []*Operation{{Action: "update", ID: restoreCacheQuotedID, Data: edit}}}); nil != err {
		t.Fatalf("edit the paragraph in the blockquote failed: %v", err)
	}
	if "quote" != cache.GetBlockIAL(restoreCacheQuoteID)["custom-shelf"] {
		t.Fatal("setup: editing the paragraph in the blockquote didn't cache the blockquote's attributes under its bare id")
	}
}

// checkRestoredPage checks that the page's file holds the restored version, and then that getBlockAttrs serves it for each
// of the page's blocks: the page's Type and its paragraph's custom-shelf, and nothing for the blockquote the restore
// removed.
func checkRestoredPage(t *testing.T, boxID, pagePath, how string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(util.DataDir, boxID, pagePath))
	if nil != err {
		t.Fatal(err)
	}
	onDisk, err := filesys.LoadTreeByData(data, boxID, pagePath, util.NewLute())
	if nil != err {
		t.Fatal(err)
	}
	paragraph := treenode.GetNodeInTree(onDisk, restoreCacheParagraphID)
	if "note" != onDisk.Root.IALAttr("custom-object-type") || nil == paragraph || "left" != paragraph.IALAttr("custom-shelf") ||
		nil != treenode.GetNodeInTree(onDisk, restoreCacheQuoteID) {
		t.Fatalf("setup: after %s, the page's file doesn't hold the restored version", how)
	}
	if got := sql.GetBlockAttrs(restoreCachePageID)["custom-object-type"]; "note" != got {
		t.Errorf("after %s, getBlockAttrs gives the page custom-object-type=%q, though its file says note", how, got)
	}
	if got := sql.GetBlockAttrs(restoreCacheParagraphID)["custom-shelf"]; "left" != got {
		t.Errorf("after %s, getBlockAttrs gives the paragraph custom-shelf=%q, though the file says left", how, got)
	}
	if got := sql.GetBlockAttrs(restoreCacheQuoteID)["custom-shelf"]; "" != got {
		t.Errorf("after %s, getBlockAttrs gives the blockquote it removed custom-shelf=%q", how, got)
	}
}

// A page restored from its history reads with the restored version's Attributes (audit #12). On SiYuan v3.8.7
// RollbackDocHistory rewrites and reindexes the page but drops none of its blocks' cached Attributes, so getBlockAttrs
// keeps serving the Attributes set after the history was taken, a removed block's too.
func TestRollbackDocHistoryRestoresAttributes(t *testing.T) {
	boxID, pagePath := setupRestoreCacheTest(t)
	// The page's history, as SiYuan keeps it: a copy of its file under the history folder, by notebook.
	historyPath := filepath.Join("history", "2026-10-16-110000-update", boxID, restoreCachePageID+".sy")
	data, err := os.ReadFile(filepath.Join(util.DataDir, boxID, pagePath))
	if nil != err {
		t.Fatal(err)
	}
	absHistoryPath := filepath.Join(util.WorkspaceDir, historyPath)
	if err = os.MkdirAll(filepath.Dir(absHistoryPath), 0755); nil != err {
		t.Fatal(err)
	}
	if err = os.WriteFile(absHistoryPath, data, 0644); nil != err {
		t.Fatal(err)
	}

	changeRestoreCachePage(t, boxID)
	if err = RollbackDocHistory(historyPath); nil != err {
		t.Fatalf("roll the page back to its history failed: %v", err)
	}
	checkRestoredPage(t, boxID, pagePath, "a page history restore")
}

// A page restored from a snapshot, one page at a time, reads with the snapshot's Attributes (audit #12). On SiYuan
// v3.8.7 RollbackRepoSnapshotFile rewrites and reindexes the page but drops none of its blocks' cached Attributes.
func TestRollbackRepoSnapshotFileRestoresAttributes(t *testing.T) {
	boxID, pagePath := setupRestoreCacheTest(t)
	originalRepoDir := util.RepoDir
	util.RepoDir = filepath.Join(util.WorkspaceDir, "repo")
	t.Cleanup(func() { util.RepoDir = originalRepoDir })
	Conf.Repo.Store(&conf.Repo{Key: bytes.Repeat([]byte{7}, 32)}) // snapshots need the repository's key
	Conf.System.Store(&conf.System{ID: "20261016110000-device1", Name: "test", OS: "linux"})
	originalStatusBar := util.StatusBarCfg.Load()
	util.StatusBarCfg.Store(util.NewStatusBar(false))
	t.Cleanup(func() { util.StatusBarCfg.Store(originalStatusBar) })

	snapshotID, err := IndexRepo("before the Type changed")
	if nil != err {
		t.Fatalf("take a snapshot failed: %v", err)
	}
	repo, err := newRepository()
	if nil != err {
		t.Fatal(err)
	}
	index, err := repo.GetIndex(snapshotID)
	if nil != err {
		t.Fatal(err)
	}
	fileID := ""
	for _, id := range index.Files {
		file, getErr := repo.GetFile(id)
		if nil != getErr {
			t.Fatal(getErr)
		}
		if "/"+boxID+pagePath == file.Path {
			fileID = id
		}
	}
	if "" == fileID {
		t.Fatalf("setup: the snapshot holds no %s", "/"+boxID+pagePath)
	}

	changeRestoreCachePage(t, boxID)
	if err = RollbackRepoSnapshotFile(fileID); nil != err {
		t.Fatalf("roll the page back to the snapshot failed: %v", err)
	}
	checkRestoredPage(t, boxID, pagePath, "a one-page snapshot restore")
}
