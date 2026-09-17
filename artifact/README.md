# Artifact

The shared renderer every skill uses to show a result as a page: one Markdown file in, one standalone Geist-styled HTML file out, opened in the browser.

```sh
./artifact/render findings.md --title "Which Node is LTS?" --eyebrow "Web research"
```

`./artifact/render --help` lists every flag. It needs Python 3 and a one-time `pnpm install` in this repo. The renderer (Marked, sanitised by DOMPurify) and the [Geist](../skills/geist/SKILL.md) token and type CSS are embedded in the output, so the page works offline in light and dark themes. Read the Geist skill before changing [`page.html`](page.html).

## Giving a skill the artifact

A skill is reached through its Link, so it cannot see this directory by path. Give it a relative symlink to the command instead:

```sh
ln -s ../../artifact/render skills/<name>/artifact
```

Then, in the skill's `SKILL.md`, have the agent write its result to a Markdown file and run `<skill-directory>/artifact <file> --title "<title>" --eyebrow "<what kind of page this is>"`. State the flags the skill needs and leave the rest to `--help`.

## Citations

When the Markdown has a `Sources` heading followed by a numbered list, the page links every `[n]` in the prose to item `n` and shows the source on hover.

`--citations` makes that a gate. The command goes red (exit 2, nothing written) and names each line where a marker has no Sources item, a source has no URL on its first line, a source is never cited, or the numbering skips. `--cited-section HEADING` also demands a marker in every paragraph, table, and list item under that heading. Fenced and inline code are ignored, and so are block quotes, which sit beside a claim rather than making one.

## Tests

```sh
bash artifact/test.sh
```
