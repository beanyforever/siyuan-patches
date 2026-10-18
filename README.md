# siyuan-patches

Small changes to [SiYuan](https://github.com/siyuan-note/siyuan), applied to an official
release and shipped as a Docker image built on the official one.

## Patches

| Patch | What it changes |
|---|---|
| `0001-mcp-custom-instructions.patch` | SiYuan's MCP server sends Settings → AI → Agent → Custom instructions (`data/ai/AGENTS.md`) to MCP clients as its `instructions`, so an agent such as Claude gets the workspace's rules when it connects. The file is read on every connection; a blank file sends nothing. |
| `0002-refuse-property-value-loss.patch` | Databases refuse edits that would destroy the values of a property, a field whose description starts with `key: <key>`, once a row fills it in: deleting it, changing its format or its key, changing what a relation property links to or its pairing, and removing or renaming an option that rows use. A primary field's Type mark `(type: …)` is kept whatever the rows hold, and deleting a page through a block deletion is refused too. Everything else passes, undoing "add option" may clear the rows that same edit picked the option on, and a field without a key, a property no row fills in yet, such as a key just typed or a copy Duplicate made, and a rollup, which holds nothing of its own, are left alone. Over the API, MCP and the CLI the reason is the error; where SiYuan reports a failed transaction to the open windows, as it does the editor's, each of them reloads, then shows it. |
| `0003-refuse-deleting-pages-that-hold-objects.patch` | Deleting a system page, one whose `custom-object-type` is `system`, is refused while pages sit under it, and so is removing a notebook that holds a system page, since either would delete those pages with it, such as every Person under People. A system page with nothing under it, and any other page, are deleted as before. The page tree, the API, MCP and the CLI all get the reason as the error; files deleted directly, a snapshot checkout and sync go around it. |
| `0004-searches-leave-blocks-whole.patch` | A search leaves the blocks it finds reading whole everywhere else in SiYuan. Without it, a full-text search on its FTS route (a keyword search that doesn't search paths, and any query-syntax search), the block-ref search typed after `((`, and opening a page from a search, which highlights its keywords, stored the rows they read in SQL's block cache, cut to a snippet or with search marks in them, and SiYuan's reads of a block by id, behind ref text in exports and templates and the backlinks panel, then served a found block cut short or marked up until it changed or the cache was cleared. Those three now read their rows without caching them; every other read of blocks caches as before. Still open: SQL's block cache changes before the SQL commit, so a read in that moment can cache the old row, and a search by SQL or an embed block whose own statement selects `snippet()` or `highlight()` in place of a block's fields still caches what it reads. |
| `0005-caches-serve-no-older-copy.patch` | SiYuan's caches of pages, of blocks' and pages' attributes and of databases serve no copy older than the file it copies. Without it, a cache could, and the next edit could save that copy over the file: of two quick puts of a new block's attributes ristretto kept the first, so "+ New" in a table left the page it made without `custom-avs` about 1 time in 3 to 6; a read that missed cached what it read even if a save landed meanwhile; a page moved to another notebook and back loaded as it left; a restored page kept its blocks' old attributes; a read of a database's history copy cached it as the live database; and a job put the attributes it changed in the cache before its commit wrote the page, leaving them there if it failed. Each page and database now has a version that rises at both ends of every write to its file, every entry carries the version noted before its source was read, and an entry a write has passed is a miss, read again from the file. A job's attribute changes enter the cache once its commit writes the page, and a failed job caches none. Still open: a writer outside SiYuan's job queue can still save a copy it read before another save over that save, which needs writers to take turns. |
| `0006-outside-writers-raise.patch` | Writers that put pages and databases on disk around SiYuan's own save functions raise what they wrote, so the caches don't serve the copy from before: the file API, the archive API's unzip, the MCP file tool, imports, notebook mount and removal, moves across notebooks, snapshot restores, sync and checkout, the user-guide mount and WebDAV. A path raises its page, its database, or everything a folder holds; a folder remove also raises whatever was written meanwhile; bulk writers raise everything. Without it, the next edit after one of those writes could save the older copy over the file, and rolling a cached page back to a snapshot it already matched deleted its file. Still open: with on-demand asset downloads, a page a sync removes stays served until the sync ends, since SiYuan's sync library removes it without telling SiYuan which file; and the CLI writes in its own process. |
| `0007-tripwire.patch` | A page or database save made outside a job's turn logs, once per call site and process, a warning naming where it came from and its caller: `tripwire: save outside a turn: <page or database> <function> <file>:<line> <- <caller>`, and each further caller of a logged site once as `tripwire: another caller of a save outside a turn: …`. Jobs hold SiYuan's flushLock while they read, change and save; a writer that saves without it can save a copy it read before a job's save over that save, so this lists the writers that should take their turn too. The first save of a new page or database and the repairs made while loading aren't logged. It is a guide, not a proof: it misses a save that lands during another writer's turn and writes around the save functions. Read the list with `grep -hE 'tripwire: (save\|another caller of a save) outside a turn: ' siyuan.log`. |

Each patch carries its tests, and the image build runs them, but for `0005`'s stress test of
jobs, reads, renders and writers at once, `TestLocksStressJobsReadsRendersAndWriters`, which
takes seconds.

## Build

```sh
docker build --build-arg SIYUAN_VERSION=v3.8.7 -t siyuan-patched:v3.8.7 .
```

Run the image exactly as the official `b3log/siyuan` one; only `/opt/siyuan/kernel` differs.

## Moving to a new SiYuan release

Build with the new `SIYUAN_VERSION`. The build stops if a patch no longer applies, compiles
or passes its test; then rebase the patches on the new tag in a SiYuan checkout and replace
the files. `git am --reject patches/<patch>` leaves each hunk that doesn't apply in a `.rej`
file to redo by hand, and `git format-patch -N` writes the patches back as `[PATCH]` each.

## Licence

SiYuan is licensed under the GNU AGPL v3, and so are these patches; see [LICENSE](LICENSE).
