package server

import (
	"strings"
	"testing"

	"github.com/itsnauman/wikiclaudia/wiki"
)

func TestRendererConvertsWikiLinks(t *testing.T) {
	renderer := NewRenderer()
	html, _, err := renderer.Render("See [[alpha-page]].", map[string]wiki.LinkTarget{
		"alpha-page": {
			Exists: true,
			Title:  "Alpha Page",
		},
	})
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	output := string(html)
	if !strings.Contains(output, `<a href="/wiki/alpha-page" class="wiki-link">Alpha Page</a>`) {
		t.Fatalf("unexpected output: %s", output)
	}
}

func TestRendererLeavesWikiLinksInsideCodeAlone(t *testing.T) {
	renderer := NewRenderer()
	body := "Inline `[[alpha-page]]` stays literal.\n\n```\n[[alpha-page]]\n```\n"
	html, _, err := renderer.Render(body, map[string]wiki.LinkTarget{
		"alpha-page": {Exists: true, Title: "Alpha Page"},
	})
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	output := string(html)
	if !strings.Contains(output, "<code>[[alpha-page]]</code>") {
		t.Fatalf("inline code should preserve literal brackets: %s", output)
	}
	if !strings.Contains(output, "<pre><code>[[alpha-page]]\n</code></pre>") {
		t.Fatalf("fenced code block should preserve literal brackets: %s", output)
	}
	if strings.Contains(output, `class="wiki-link"`) {
		t.Fatalf("wiki-link class should not appear inside code regions: %s", output)
	}
}

func TestRendererMarksMissingWikiLinks(t *testing.T) {
	renderer := NewRenderer()
	html, _, err := renderer.Render("See [[missing-page]].", map[string]wiki.LinkTarget{
		"missing-page": {
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

func TestRendererCustomLinkText(t *testing.T) {
	renderer := NewRenderer()
	html, _, err := renderer.Render("See [[strength-based-model|strengths-based]].", map[string]wiki.LinkTarget{
		"strength-based-model": {
			Exists: true,
			Title:  "Strength Based Model",
		},
	})
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	output := string(html)
	if !strings.Contains(output, `<a href="/wiki/strength-based-model" class="wiki-link">strengths-based</a>`) {
		t.Fatalf("custom link text not rendered correctly: %s", output)
	}
}

func TestRendererCustomLinkTextMissingPage(t *testing.T) {
	renderer := NewRenderer()
	html, _, err := renderer.Render("See [[no-such-page|click here]].", map[string]wiki.LinkTarget{
		"no-such-page": {
			Exists: false,
			Title:  "No Such Page",
		},
	})
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	output := string(html)
	if !strings.Contains(output, `<a href="/wiki/no-such-page" class="wiki-link missing">click here</a>`) {
		t.Fatalf("custom link text with missing page not rendered correctly: %s", output)
	}
}

func TestCollectWikiLinkSlugsWithPipedLinks(t *testing.T) {
	slugs := collectWikiLinkSlugs("See [[alpha|custom text]] and [[beta-page]].")
	if len(slugs) != 2 {
		t.Fatalf("expected 2 slugs, got %d: %v", len(slugs), slugs)
	}
	if slugs[0] != "alpha" {
		t.Fatalf("expected slug 'alpha', got %q", slugs[0])
	}
	if slugs[1] != "beta-page" {
		t.Fatalf("expected slug 'beta-page', got %q", slugs[1])
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
