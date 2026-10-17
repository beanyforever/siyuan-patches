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

Each patch carries a test, and the image build runs it.

## Acceptance tests

SiYuan's page cache, its block and page attribute caches and its database cache can each
serve a copy older than the file it copies, and the next edit can save that copy over the
file. No patch here fixes that yet: the patches that closed some of the ways in, one at a
time, were taken out for one fix at the root (polka-siyuan's issue #43), so the image caches
as the official build does. `acceptance/` holds their Go tests of those faults, which fail,
or don't compile, on the image's source, and which the build doesn't run; a patch that turns
one green moves it into the build. [acceptance/README.md](acceptance/README.md) lists them and how to run them.

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
