# wikiclaudia

A local Wikipedia-style reader for LLM-maintained personal wikis.

`wikiclaudia` is a single-binary Go CLI that serves a directory of markdown pages as an interlinked HTML site in your browser. It reads wikis produced by [`wiki-skills`](https://github.com/itsnauman/wiki-skills), a Claude Code plugin that implements [Andrej Karpathy's LLM Wiki pattern](https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f): you feed an LLM sources and ask it questions, and it grows a cross-linked knowledge base as it goes.

The wiki itself is written by the LLM. `wikiclaudia` is the viewer.

## Features

- **Zero config.** `cd` into a wiki directory and run `wikiclaudia`. It checks the layout, starts an HTTP server on `127.0.0.1:8080`, and opens your browser.
- **Wikipedia Vector 2022 styling.** Article-first layout, sidebar table of contents, metadata chips for tags and sources, and a light/dark toggle that follows your system preference.
- **`[[wiki-link]]` syntax.** Double-bracket slugs resolve to real links at render time. Targets that don't exist yet show up as red stub links, the way Wikipedia does it.
- **Frontmatter-aware.** Title, tags, sources, and last-updated date come from YAML frontmatter and show up in a metadata panel. Sources link to other pages.
- **GitHub-Flavored Markdown** via [goldmark](https://github.com/yuin/goldmark), with auto-generated heading IDs and a table of contents.
- **Self-contained.** Templates and CSS are embedded in the binary. Nothing else to install or host.

## Install

```sh
go install github.com/itsnauman/wikiclaudia@latest
```

Requires Go 1.22 or newer. The binary lands at `$(go env GOBIN)/wikiclaudia` (or `$GOPATH/bin/wikiclaudia`), so make sure that directory is on your `PATH`.

## Usage

From the root of a wiki directory:

```sh
wikiclaudia
```

Flags:

| Flag    | Default     | Description                  |
| ------- | ----------- | ---------------------------- |
| `-host` | `127.0.0.1` | host to bind                 |
| `-port` | `8080`      | port to bind                 |

Example:

```sh
wikiclaudia -host 0.0.0.0 -port 9000
```

`wikiclaudia` prints the serve URL to stdout and tries to open it in your default browser. `Ctrl+C` to stop.

## Expected wiki layout

`wikiclaudia` expects the directory structure produced by [`wiki-skills`](https://github.com/itsnauman/wiki-skills):

```
<wiki-root>/
├── SCHEMA.md           # Wiki identity and conventions
├── raw/                # Immutable source documents (not served)
├── assets/             # Images and attachments, served at /assets/*
└── wiki/
    ├── index.md        # Home page, served at /
    ├── log.md          # Append-only operation ledger (not served)
    ├── overview.md     # Synthesized summary, served at /overview
    └── pages/          # Flat directory of slug-named pages
        ├── alpha-page.md
        └── beta-page.md
```

On startup, `wikiclaudia` checks that the required files exist, parses `SCHEMA.md`, and validates the frontmatter of `overview.md` and every file under `wiki/pages/`. If anything is missing or malformed, it exits with a descriptive error before binding a port.

### `SCHEMA.md`

Must contain at least these two lines:

```markdown
- **Path:** /absolute/path/to/this/wiki
- **Domain:** My Knowledge Domain
```

The `Path` must resolve to the current working directory. That check exists so you don't accidentally run `wikiclaudia` against the wrong wiki.

### Page frontmatter

`overview.md` and every page under `wiki/pages/` must start with a frontmatter block containing all four fields:

```markdown
---
title: Alpha Page
tags: [alpha, testing]
sources: [source-entry, another-source]
updated: 2026-04-07
---

# Alpha Page

Body text with a [[beta-page]] link.
```

- `title` — shown in `<title>` and the metadata panel
- `tags` — rendered as chips
- `sources` — slugs of other pages; shown as clickable links in the metadata panel
- `updated` — `YYYY-MM-DD`

`index.md` may include frontmatter but doesn't have to.

### Wiki links

Any `[[slug]]` inside body text (outside of code spans and fenced code blocks) becomes a link to `/wiki/<slug>`. If the page exists, the link label comes from its frontmatter title. If it doesn't, the label is a humanized version of the slug and the link is styled as missing.

## Building a wiki

`wikiclaudia` is just the reader. It doesn't create or edit wiki content. To grow a wiki from your sources, install [`wiki-skills`](https://github.com/itsnauman/wiki-skills) in Claude Code and use its `wiki-init`, `wiki-ingest`, `wiki-query`, `wiki-update`, and `wiki-lint` commands.

The loop:

1. **Write** — Use `wiki-skills` in Claude Code to ingest sources and grow your wiki.
2. **Read** — Run `wikiclaudia` in the same directory to browse the result.

## Credits

- [Andrej Karpathy](https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f) — for the LLM Wiki pattern that motivated this tool.
- [`kfchou/wiki-skills`](https://github.com/kfchou/wiki-skills) — the original Claude Code plugin implementing the LLM Wiki pattern.
- [goldmark](https://github.com/yuin/goldmark) — the markdown parser and renderer.
