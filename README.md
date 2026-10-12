# siyuan-patches

Small changes to [SiYuan](https://github.com/siyuan-note/siyuan), applied to an official
release and shipped as a Docker image built on the official one.

## Patches

| Patch | What it changes |
|---|---|
| `0001-mcp-custom-instructions.patch` | SiYuan's MCP server sends Settings → AI → Agent → Custom instructions (`data/ai/AGENTS.md`) to MCP clients as its `instructions`, so an agent such as Claude gets the workspace's rules when it connects. The file is read on every connection; a blank file sends nothing. |
| `0002-refuse-property-value-loss.patch` | Databases refuse edits that would destroy the values of a property, a field whose description starts with `key: <key>`: deleting it, changing its format, its key or a primary field's Type mark `(type: …)`, changing what a relation property links to or its pairing, and removing or renaming an option that rows use. Deleting a page through a block deletion is refused too. Everything else passes, undoing "add option" may clear the rows that same edit picked the option on, and a field without a key is left alone. Over the API, MCP and the CLI the reason is the error; in the editor every open window reloads, then shows it. |

Each patch carries a test, and the image build runs it.

## Build

```sh
docker build --build-arg SIYUAN_VERSION=v3.8.6 -t siyuan-patched:v3.8.6 .
```

Run the image exactly as the official `b3log/siyuan` one; only `/opt/siyuan/kernel` differs.

## Moving to a new SiYuan release

Build with the new `SIYUAN_VERSION`. The build stops if a patch no longer applies or its
test fails; then rebase the patch on the new tag in a SiYuan checkout
(`git am patches/*.patch`, fix, `git format-patch`) and replace the file.

## Licence

SiYuan is licensed under the GNU AGPL v3, and so are these patches; see [LICENSE](LICENSE).
