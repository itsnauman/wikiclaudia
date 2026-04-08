package server

import (
	"strings"
	"testing"

	"wikiclaudia/wiki"
)

func TestRendererConvertsWikiLinks(t *testing.T) {
	renderer := NewRenderer()
	html, _, err := renderer.Render("See [[alpha-page]].", map[string]wiki.LinkTarget{
		"alpha-page": {
			Slug:   "alpha-page",
			Exists: true,
			Title:  "Alpha Page",
		},
	})
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	output := string(html)
	if !strings.Contains(output, `<a class="wiki-link" href="/wiki/alpha-page">Alpha Page</a>`) {
		t.Fatalf("unexpected output: %s", output)
	}
}

func TestRendererMarksMissingWikiLinks(t *testing.T) {
	renderer := NewRenderer()
	html, _, err := renderer.Render("See [[missing-page]].", map[string]wiki.LinkTarget{
		"missing-page": {
			Slug:   "missing-page",
			Exists: false,
			Title:  "Missing Page",
		},
	})
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	output := string(html)
	if !strings.Contains(output, `class="wiki-link missing"`) {
		t.Fatalf("missing page link not marked correctly: %s", output)
	}
	if !strings.Contains(output, `href="/wiki/missing-page"`) {
		t.Fatalf("missing page link has wrong target: %s", output)
	}
}

func TestRendererPreservesMarkdownFormatting(t *testing.T) {
	renderer := NewRenderer()
	body := "# Title\n\nA *paragraph* with **strong** text.\n\n> Quoted note.\n\n- One\n- Two\n"
	html, _, err := renderer.Render(body, nil)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	output := string(html)
	for _, fragment := range []string{
		"<em>paragraph</em>",
		"<strong>strong</strong>",
		"<blockquote>",
		"<ul>",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("expected fragment %q in output %s", fragment, output)
		}
	}
}

func TestRendererGeneratesHeadingIDsAndTOC(t *testing.T) {
	renderer := NewRenderer()
	body := "# Title\n\n## First Section\n\n### Deep Dive\n\n## First Section\n"
	html, toc, err := renderer.Render(body, nil)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	output := string(html)
	for _, fragment := range []string{
		`<h2 id="first-section">First Section</h2>`,
		`<h3 id="deep-dive">Deep Dive</h3>`,
		`<h2 id="first-section-1">First Section</h2>`,
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("expected fragment %q in output %s", fragment, output)
		}
	}

	if len(toc) != 3 {
		t.Fatalf("expected 3 toc entries, got %d", len(toc))
	}

	if toc[0].ID != "first-section" || toc[1].ID != "deep-dive" || toc[2].ID != "first-section-1" {
		t.Fatalf("unexpected toc ids: %#v", toc)
	}
}
