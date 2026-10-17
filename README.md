# siyuan-patches

Small changes to [SiYuan](https://github.com/siyuan-note/siyuan), applied to an official
release and shipped as a Docker image built on the official one.

## Patches

| Patch | What it changes |
|---|---|
| `0001-mcp-custom-instructions.patch` | SiYuan's MCP server sends Settings → AI → Agent → Custom instructions (`data/ai/AGENTS.md`) to MCP clients as its `instructions`, so an agent such as Claude gets the workspace's rules when it connects. The file is read on every connection; a blank file sends nothing. |
| `0002-refuse-property-value-loss.patch` | Databases refuse edits that would destroy the values of a property, a field whose description starts with `key: <key>`, once a row fills it in: deleting it, changing its format or its key, changing what a relation property links to or its pairing, and removing or renaming an option that rows use. A primary field's Type mark `(type: …)` is kept whatever the rows hold, and deleting a page through a block deletion is refused too. Everything else passes, undoing "add option" may clear the rows that same edit picked the option on, and a field without a key, a property no row fills in yet, such as a key just typed or a copy Duplicate made, and a rollup, which holds nothing of its own, are left alone. Over the API, MCP and the CLI the reason is the error; where SiYuan reports a failed transaction to the open windows, as it does the editor's, each of them reloads, then shows it. |
| `0003-refuse-deleting-pages-that-hold-objects.patch` | Deleting a system page, one whose `custom-object-type` is `system`, is refused while pages sit under it, and so is removing a notebook that holds a system page, since either would delete those pages with it, such as every Person under People. A system page with nothing under it, and any other page, are deleted as before. The page tree, the API, MCP and the CLI all get the reason as the error; files deleted directly, a snapshot checkout and sync go around it. |
| `0004-attribute-caches-keep-newest.patch` | A read of a block's or a page's attributes gives the newest ones the cache holds, however close together two writes come: "+ New" in a database no longer leaves the page it makes with its attributes from before it was bound, without `custom-avs`, so that `/api/attr/getBlockAttrs` and `/api/av/getAttributeViewKeys` named no database for it until SiYuan restarted. A write waits until the cache has taken it in or turned it away, and one turned away removes the block's or page's older copies, so reads go to the file. A read that missed the cache caches what it loaded only if nothing wrote the block, or the page it read the block from, since the read began. Still open: a write whose entry is gone before a later read inside the same transaction, a failed transaction's writes, which stay cached, the `custom-avs` an import or a database duplicate can cache without the file getting it, and a notebook-key entry hiding a newer edit's bare-key copy. |
| `0005-moves-drop-stale-caches.patch` | A page moved to another notebook and back keeps every edit, attribute and block it was given while away, and a page moved after an edit reads with the attributes its file holds. Without it, SiYuan's page and block attribute caches keep a moved page's entries under the notebook it left, so back there the page and its sub-pages load as they were when they left: `/api/attr/getBlockAttrs` serves the old attributes, the block index loses the blocks added while away, and the next edit saves the old copy over the file. Moving now drops those entries for the page and each sub-page, on every way to move a page across notebooks, before the page is loaded under its new notebook; SiYuan's command-line move runs in its own process and still leaves them. |

Each patch carries a test, and the image build runs it.

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
