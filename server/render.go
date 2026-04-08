package server

import (
	"bytes"
	"fmt"
	stdhtml "html"
	"html/template"
	"net/url"
	"regexp"
	"strings"
	"unicode"

	"wikiclaudia/wiki"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
)

var wikiLinkPattern = regexp.MustCompile(`\[\[([^\[\]\r\n]+)\]\]`)

type Renderer struct {
	markdown goldmark.Markdown
}

type TOCEntry struct {
	Level int
	Text  string
	ID    string
}

func NewRenderer() *Renderer {
	return &Renderer{
		markdown: goldmark.New(
			goldmark.WithExtensions(extension.GFM),
			goldmark.WithRendererOptions(gmhtml.WithUnsafe()),
			goldmark.WithParserOptions(parser.WithAttribute(), parser.WithAutoHeadingID()),
		),
	}
}

func (r *Renderer) Render(markdown string, targets map[string]wiki.LinkTarget) (template.HTML, []TOCEntry, error) {
	processed := rewriteWikiLinks(markdown, targets)
	source := []byte(processed)
	document := r.markdown.Parser().Parse(text.NewReader(source))
	toc := assignHeadingIDs(document, source)

	var output bytes.Buffer
	if err := r.markdown.Renderer().Render(&output, source, document); err != nil {
		return "", nil, fmt.Errorf("render markdown: %w", err)
	}

	return template.HTML(output.String()), toc, nil
}

func collectWikiLinkSlugs(markdown string) []string {
	matches := wikiLinkPattern.FindAllStringSubmatch(markdown, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(matches))
	slugs := make([]string, 0, len(matches))
	for _, match := range matches {
		slug := strings.TrimSpace(match[1])
		if slug == "" {
			continue
		}
		if _, ok := seen[slug]; ok {
			continue
		}
		seen[slug] = struct{}{}
		slugs = append(slugs, slug)
	}

	return slugs
}

func rewriteWikiLinks(markdown string, targets map[string]wiki.LinkTarget) string {
	return wikiLinkPattern.ReplaceAllStringFunc(markdown, func(match string) string {
		parts := wikiLinkPattern.FindStringSubmatch(match)
		if len(parts) != 2 {
			return match
		}

		slug := strings.TrimSpace(parts[1])
		target, ok := targets[slug]
		if !ok {
			target = wiki.LinkTarget{
				Slug:  slug,
				Title: wiki.HumanizeSlug(slug),
			}
		}

		className := "wiki-link"
		if !target.Exists {
			className += " missing"
		}

		label := target.Title
		if label == "" {
			label = wiki.HumanizeSlug(slug)
		}

		return fmt.Sprintf(
			`<a class="%s" href="/wiki/%s">%s</a>`,
			className,
			url.PathEscape(slug),
			stdhtml.EscapeString(label),
		)
	})
}

func assignHeadingIDs(document ast.Node, source []byte) []TOCEntry {
	ids := newHeadingIDs()
	toc := make([]TOCEntry, 0, 8)

	_ = ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		heading, ok := node.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}

		text := strings.TrimSpace(extractText(heading, source))
		id := ids.Next(text)
		heading.SetAttribute([]byte("id"), []byte(id))

		if heading.Level > 1 {
			toc = append(toc, TOCEntry{
				Level: heading.Level,
				Text:  text,
				ID:    id,
			})
		}

		return ast.WalkContinue, nil
	})

	return toc
}

func extractText(node ast.Node, source []byte) string {
	var builder strings.Builder

	var walk func(ast.Node)
	walk = func(current ast.Node) {
		switch n := current.(type) {
		case *ast.Text:
			builder.Write(n.Segment.Value(source))
			if n.HardLineBreak() || n.SoftLineBreak() {
				builder.WriteByte(' ')
			}
		case *ast.String:
			builder.Write(n.Value)
		default:
			for child := current.FirstChild(); child != nil; child = child.NextSibling() {
				walk(child)
			}
		}
	}

	walk(node)
	return builder.String()
}

type headingIDs struct {
	counts map[string]int
}

func newHeadingIDs() *headingIDs {
	return &headingIDs{
		counts: make(map[string]int),
	}
}

func (h *headingIDs) Next(text string) string {
	base := slugifyHeading(text)
	count := h.counts[base]
	h.counts[base] = count + 1
	if count == 0 {
		return base
	}
	return fmt.Sprintf("%s-%d", base, count)
}

func slugifyHeading(text string) string {
	text = strings.TrimSpace(strings.ToLower(text))
	if text == "" {
		return "section"
	}

	var builder strings.Builder
	lastHyphen := false
	for _, r := range text {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(r)
			lastHyphen = false
		case !lastHyphen:
			builder.WriteByte('-')
			lastHyphen = true
		}
	}

	slug := strings.Trim(builder.String(), "-")
	if slug == "" {
		return "section"
	}
	return slug
}
