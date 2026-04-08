package testfixture

import (
	"fmt"
	"os"
	"path/filepath"
)

type Options struct {
	MissingRequired            string
	NestedPageDir              bool
	InvalidOverviewFrontmatter bool
	InvalidPageFrontmatter     bool
	SchemaPath                 string
}

func WriteMinimalWiki(root string, opts Options) error {
	directories := []string{
		filepath.Join(root, "assets"),
		filepath.Join(root, "raw"),
		filepath.Join(root, "wiki"),
		filepath.Join(root, "wiki", "pages"),
	}

	for _, directory := range directories {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			return fmt.Errorf("create %s: %w", directory, err)
		}
	}

	schemaPath := opts.SchemaPath
	if schemaPath == "" {
		schemaPath = root
	}

	files := map[string]string{
		"SCHEMA.md": fmt.Sprintf(`# Wiki Schema

## Identity
- **Path:** %s
- **Domain:** Test Domain
- **Source types:** Books
- **Created:** 2026-04-07
`, schemaPath),
		filepath.Join("raw", "source.md"):      "# Raw Source\n",
		filepath.Join("assets", "diagram.txt"): "asset-body\n",
		filepath.Join("wiki", "log.md"):        "## [2026-04-07] init | Test Wiki\n",
		filepath.Join("wiki", "index.md"):      "# Home\n\nSee [[alpha-page]] and [[missing-page]].\n",
		filepath.Join("wiki", "overview.md"): `---
title: Overview
tags: [overview, synthesis]
sources: [source-entry]
updated: 2026-04-07
---

# Overview

Intro with [[alpha-page]].

## Shared Themes

Overview details.
`,
		filepath.Join("wiki", "pages", "alpha-page.md"): `---
title: Alpha Page
tags: [alpha, testing]
sources: [source-entry]
updated: 2026-04-07
---

# Alpha Page

Original text with [[beta-page]] and [[missing-page]].

## Section One

A *paragraph* with **strong** text.

> Quoted note.

- Item one
- Item two
`,
		filepath.Join("wiki", "pages", "beta-page.md"): `---
title: Beta Page
tags: [beta]
sources: []
updated: 2026-04-07
---

# Beta Page

Second page.
`,
		filepath.Join("wiki", "pages", "source-entry.md"): `---
title: Source Entry
tags: [source]
sources: []
updated: 2026-04-07
---

# Source Entry

Bibliographic note.
`,
	}

	for relativePath, content := range files {
		targetPath := filepath.Join(root, relativePath)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return fmt.Errorf("create parent for %s: %w", targetPath, err)
		}
		if err := os.WriteFile(targetPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", targetPath, err)
		}
	}

	if opts.InvalidOverviewFrontmatter {
		overviewPath := filepath.Join(root, "wiki", "overview.md")
		content := `---
title: Overview
sources: [source-entry]
updated: 2026-04-07
---

# Overview
`
		if err := os.WriteFile(overviewPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write invalid overview: %w", err)
		}
	}

	if opts.InvalidPageFrontmatter {
		pagePath := filepath.Join(root, "wiki", "pages", "alpha-page.md")
		content := `---
title: Alpha Page
tags: [alpha]
updated: 2026-04-07
---

# Alpha Page
`
		if err := os.WriteFile(pagePath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write invalid page: %w", err)
		}
	}

	if opts.NestedPageDir {
		nestedDirectory := filepath.Join(root, "wiki", "pages", "nested")
		if err := os.MkdirAll(nestedDirectory, 0o755); err != nil {
			return fmt.Errorf("create nested page directory: %w", err)
		}
		nestedPagePath := filepath.Join(nestedDirectory, "child.md")
		if err := os.WriteFile(nestedPagePath, []byte("# Nested"), 0o644); err != nil {
			return fmt.Errorf("write nested page: %w", err)
		}
	}

	if opts.MissingRequired != "" {
		if err := os.RemoveAll(filepath.Join(root, opts.MissingRequired)); err != nil {
			return fmt.Errorf("remove %s: %w", opts.MissingRequired, err)
		}
	}

	return nil
}
