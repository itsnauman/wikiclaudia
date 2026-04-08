package wiki

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

var ErrPageNotFound = errors.New("page not found")

type Schema struct {
	IdentityPath string
	Domain       string
}

type Site struct {
	Root   string
	Schema Schema
}

type Article struct {
	Slug       string
	SourcePath string
	Meta       *Frontmatter
	Body       string
}

type LinkTarget struct {
	Slug   string
	Exists bool
	Title  string
}

func ValidateRoot(root string) (*Site, error) {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}

	if err := validateRequiredEntries(absoluteRoot); err != nil {
		return nil, err
	}

	schema, err := loadSchema(filepath.Join(absoluteRoot, "SCHEMA.md"))
	if err != nil {
		return nil, err
	}

	if normalizePath(schema.IdentityPath) != normalizePath(absoluteRoot) {
		return nil, fmt.Errorf("SCHEMA.md path %q does not match current directory %q", schema.IdentityPath, absoluteRoot)
	}

	if _, _, err := loadRequiredArticle(filepath.Join(absoluteRoot, "wiki", "overview.md"), "overview"); err != nil {
		return nil, fmt.Errorf("validate overview.md: %w", err)
	}

	if err := validatePages(filepath.Join(absoluteRoot, "wiki", "pages")); err != nil {
		return nil, err
	}

	return &Site{
		Root:   absoluteRoot,
		Schema: schema,
	}, nil
}

func LoadIndex(root string) (*Article, error) {
	path := filepath.Join(root, "wiki", "index.md")
	meta, body, err := loadOptionalArticle(path)
	if err != nil {
		return nil, fmt.Errorf("load index.md: %w", err)
	}

	return &Article{
		Slug:       "",
		SourcePath: path,
		Meta:       meta,
		Body:       string(body),
	}, nil
}

func LoadOverview(root string) (*Article, error) {
	path := filepath.Join(root, "wiki", "overview.md")
	meta, body, err := loadRequiredArticle(path, "overview")
	if err != nil {
		return nil, fmt.Errorf("load overview.md: %w", err)
	}

	return &Article{
		Slug:       "overview",
		SourcePath: path,
		Meta:       meta,
		Body:       string(body),
	}, nil
}

func LoadPage(root, slug string) (*Article, error) {
	if err := validateSlug(slug); err != nil {
		return nil, err
	}

	path := filepath.Join(root, "wiki", "pages", slug+".md")
	meta, body, err := loadRequiredArticle(path, slug)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrPageNotFound
		}
		return nil, fmt.Errorf("load page %q: %w", slug, err)
	}

	return &Article{
		Slug:       slug,
		SourcePath: path,
		Meta:       meta,
		Body:       string(body),
	}, nil
}

func ResolveLinks(root string, slugs []string) map[string]LinkTarget {
	targets := make(map[string]LinkTarget, len(slugs))
	seen := make(map[string]struct{}, len(slugs))

	for _, slug := range slugs {
		slug = strings.TrimSpace(slug)
		if slug == "" {
			continue
		}
		if _, ok := seen[slug]; ok {
			continue
		}
		seen[slug] = struct{}{}

		target := LinkTarget{
			Slug:  slug,
			Title: HumanizeSlug(slug),
		}

		if err := validateSlug(slug); err == nil {
			path := filepath.Join(root, "wiki", "pages", slug+".md")
			info, err := os.Stat(path)
			if err == nil && info.Mode().IsRegular() {
				target.Exists = true

				content, readErr := os.ReadFile(path)
				if readErr == nil {
					if block, _, has, splitErr := SplitFrontmatter(content); splitErr == nil && has {
						if meta, parseErr := ParseFrontmatter(block); parseErr == nil && meta.Title != "" {
							target.Title = meta.Title
						}
					}
				}
			}
		}

		targets[slug] = target
	}

	return targets
}

func HumanizeSlug(slug string) string {
	parts := strings.FieldsFunc(slug, func(r rune) bool {
		return r == '-' || r == '_' || unicode.IsSpace(r)
	})
	if len(parts) == 0 {
		return slug
	}

	for i, part := range parts {
		runes := []rune(strings.ToLower(part))
		if len(runes) == 0 {
			continue
		}
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}

	return strings.Join(parts, " ")
}

func validateRequiredEntries(root string) error {
	requiredFiles := []string{
		"SCHEMA.md",
		filepath.Join("wiki", "index.md"),
		filepath.Join("wiki", "log.md"),
		filepath.Join("wiki", "overview.md"),
	}
	requiredDirs := []string{
		"raw",
		"assets",
		filepath.Join("wiki", "pages"),
	}

	for _, relativePath := range requiredFiles {
		if err := ensureReadableFile(filepath.Join(root, relativePath)); err != nil {
			return err
		}
	}

	for _, relativePath := range requiredDirs {
		if err := ensureDirectory(filepath.Join(root, relativePath)); err != nil {
			return err
		}
	}

	return nil
}

func validatePages(pagesDir string) error {
	entries, err := os.ReadDir(pagesDir)
	if err != nil {
		return fmt.Errorf("read wiki/pages: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			return fmt.Errorf("wiki/pages must be flat; found subdirectory %q", name)
		}
		if filepath.Ext(name) != ".md" {
			return fmt.Errorf("wiki/pages may only contain .md files; found %q", name)
		}

		path := filepath.Join(pagesDir, name)
		if _, _, err := loadRequiredArticle(path, strings.TrimSuffix(name, ".md")); err != nil {
			return fmt.Errorf("validate %s: %w", name, err)
		}
	}

	return nil
}

func loadRequiredArticle(path string, label string) (*Frontmatter, []byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	meta, body, err := ParseRequiredFrontmatter(content)
	if err != nil {
		return nil, nil, fmt.Errorf("parse %s frontmatter: %w", label, err)
	}

	return meta, body, nil
}

func loadOptionalArticle(path string) (*Frontmatter, []byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	return parseOptionalFrontmatter(content)
}

func ensureReadableFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("missing required file %q", path)
		}
		return fmt.Errorf("stat file %q: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("required file %q is not a regular file", path)
	}
	if _, err := os.ReadFile(path); err != nil {
		return fmt.Errorf("read required file %q: %w", path, err)
	}
	return nil
}

func ensureDirectory(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("missing required directory %q", path)
		}
		return fmt.Errorf("stat directory %q: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("required directory %q is not a directory", path)
	}
	return nil
}

func loadSchema(path string) (Schema, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Schema{}, fmt.Errorf("read SCHEMA.md: %w", err)
	}

	var schema Schema
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "- **Path:** "):
			schema.IdentityPath = strings.TrimSpace(strings.TrimPrefix(line, "- **Path:** "))
		case strings.HasPrefix(line, "- **Domain:** "):
			schema.Domain = strings.TrimSpace(strings.TrimPrefix(line, "- **Domain:** "))
		}
	}

	if schema.IdentityPath == "" {
		return Schema{}, errors.New("SCHEMA.md is missing Identity Path")
	}
	if schema.Domain == "" {
		return Schema{}, errors.New("SCHEMA.md is missing Domain")
	}

	return schema, nil
}

func normalizePath(path string) string {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		absolutePath = path
	}
	if resolved, err := filepath.EvalSymlinks(absolutePath); err == nil {
		absolutePath = resolved
	}
	return filepath.Clean(absolutePath)
}

func validateSlug(slug string) error {
	switch {
	case slug == "":
		return fmt.Errorf("slug is required")
	case filepath.Base(slug) != slug:
		return fmt.Errorf("invalid slug %q", slug)
	case strings.Contains(slug, ".."):
		return fmt.Errorf("invalid slug %q", slug)
	case strings.ContainsAny(slug, `/\`):
		return fmt.Errorf("invalid slug %q", slug)
	default:
		return nil
	}
}
