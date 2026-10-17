# Acceptance tests

Go tests of faults in SiYuan's page, block and page attribute, and database caches that no
patch in `patches/` fixes yet: a cache serves a copy older than its file, and the next edit
can save that copy over the file. They come from the earlier patches that closed those faults
one way in at a time, taken out for one fix at the root (polka-siyuan's issue #43). The image
build doesn't run them.

On the image's source, SiYuan v3.8.7 with `patches/` applied, each test fails, or its file
doesn't compile because it calls code a later patch brings in. The patch that turns a test
green moves it into its own patch and into the Dockerfile's test run, and removes it here.

## Tests

The sorts:

- **keep as is:** it passes unchanged once the fault is fixed.
- **keep, setup rewritten:** its setup checks the cache directly to confirm the fault, which
  stops it once the fault is fixed, so the setup is rewritten before it joins the build.
- **keep:** it calls code the fix reuses from the earlier patch, and passes once that's back.
- **rewrite:** it is rewritten before it joins the build: against the fix, where it tests the
  earlier patch's own mechanism, or to wait for the cache, where marked *.

| Test | File | From | Sort | On the image's source |
|---|---|---|---|---|
| `TestMoveToAnotherNotebookAndBackKeepsEdits` | `kernel/model/move_cache_fts5_test.go` | 0005 moves drop stale caches | keep as is | fails |
| `TestMoveAfterEditReadsNewType` | `kernel/model/move_cache_fts5_test.go` | 0005 | keep as is | fails |
| `TestTreeFillRacesWrite` | `kernel/filesys/tree_fill_race_test.go` | 0006 page cache fills keep newest | keep as is | fails |
| `TestRollbackDocHistoryRestoresAttributes` | `kernel/model/restore_cache_fts5_test.go` | 0007 restores drop stale attributes | keep, setup rewritten | fails |
| `TestRollbackRepoSnapshotFileRestoresAttributes` | `kernel/model/restore_cache_fts5_test.go` | 0007 | keep, setup rewritten | fails |
| `TestFillVersionsForgetPastTheirLimit` | `kernel/cache/ial_fill_test.go` | 0004 attribute caches keep newest | keep | doesn't compile |
| `TestBatchGetBlockAttrsWitTreesCachesNothing` | `kernel/sql/ial_fill_test.go` | 0004 | rewrite* | doesn't compile |
| `TestBlockAttrsReadsFillWhatTheyLoad` | `kernel/sql/ial_fill_test.go` | 0004 | rewrite* | doesn't compile |
| `TestIALCacheKeepsLatestPut` | `kernel/cache/ial_latest_test.go` | 0004 | rewrite | fails |
| `TestIALBlockFillChecksItsPage` | `kernel/cache/ial_fill_test.go` | 0004 | rewrite | doesn't compile |
| `TestWriteTreeRefusesBlockFillsTakenBefore` | `kernel/filesys/ial_fill_test.go` | 0004 | rewrite | doesn't compile |
| `TestBlockAttrsFillThenWriteKeepsWrite` | `kernel/sql/ial_fill_test.go` | 0004 | rewrite | doesn't compile |
| `TestTreeFillsDuringFileWriteCacheNothing` | `kernel/filesys/tree_fill_during_write_test.go` | 0006 | rewrite | doesn't compile |
| `TestAVCacheReadAfterSaveHits` | `kernel/cache/av_save_test.go` | 0008 database saves readable at once | rewrite | fails |

\* Their file doesn't compile for `TestBlockAttrsFillThenWriteKeepsWrite`'s sake. On their own,
each reads the cache straight after a read that stores to it, before the cache has taken the
store in: so `TestBatchGetBlockAttrsWitTreesCachesNothing` passes though the read caches what it
read, and `TestBlockAttrsReadsFillWhatTheyLoad` fails though it does.

## Running them

Copy the folder into a checkout of the release with the patches applied, then run a test from
`kernel/`:

```sh
git clone --branch v3.8.7 https://github.com/siyuan-note/siyuan.git
cd siyuan
git am <siyuan-patches>/patches/*.patch
cp -r <siyuan-patches>/acceptance/kernel/. kernel/
cd kernel
go test -tags "fts5 sqlcipher" -count=1 -run '^TestMoveAfterEditReadsNewType$' ./model/
```

A file that doesn't compile stops every test in its package, SiYuan's own among them, so copy
only the files a run needs. The `model` tests need the `fts5` tag.
