# siyuan-patches

Small changes to [SiYuan](https://github.com/siyuan-note/siyuan), applied to an official
release and shipped as a Docker image built on the official one.

## Patches

| Patch | What it changes |
|---|---|
| `0001-mcp-custom-instructions.patch` | SiYuan's MCP server sends Settings → AI → Agent → Custom instructions (`data/ai/AGENTS.md`) to MCP clients as its `instructions`, so an agent such as Claude gets the workspace's rules when it connects. The file is read on every connection; a blank file sends nothing. |
| `0002-refuse-property-value-loss.patch` | Databases refuse edits that would destroy the values of a property, a field whose description starts with `key: <key>`, once a row fills it in: deleting it, changing its format or its key, changing what a relation property links to or its pairing, and removing or renaming an option that rows use. A primary field's Type mark `(type: …)` is kept whatever the rows hold, and deleting a page through a block deletion is refused too. Everything else passes, undoing "add option" may clear the rows that same edit picked the option on, and a field without a key, a property no row fills in yet, such as a key just typed or a copy Duplicate made, and a rollup, which holds nothing of its own, are left alone. Over the API, MCP and the CLI the reason is the error; where SiYuan reports a failed transaction to the open windows, as it does the editor's, each of them reloads, then shows it. |
| `0003-refuse-deleting-pages-that-hold-objects.patch` | Deleting a system page, one whose `custom-object-type` is `system`, is refused while pages sit under it, and so is removing a notebook that holds a system page, since either would delete those pages with it, such as every Person under People. A system page with nothing under it, and any other page, are deleted as before. The page tree, the API, MCP and the CLI all get the reason as the error; files deleted directly, a snapshot checkout and sync go around it. |
| `0004-ial-cache-keeps-latest-put.patch` | A read of a block's or a document's attributes gives the ones put last, however close together two puts come. Without it, SiYuan's cache can keep the first of two quick puts, so a page "+ New" makes a database row keeps its attributes from before it was bound, without `custom-avs`, until SiYuan restarts: `/api/attr/getBlockAttrs` and `/api/av/getAttributeViewKeys` then name no database for it, though the database, the page's file and SQL hold the binding. |

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
