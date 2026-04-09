package wiki

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const dateLayout = "2006-01-02"

type Frontmatter struct {
	Title   string
	Tags    []string
	Sources []string
	Updated time.Time
}

func (f Frontmatter) UpdatedString() string {
	return f.Updated.Format(dateLayout)
}

func SplitFrontmatter(content []byte) (string, []byte, bool, error) {
	if len(content) == 0 {
		return "", content, false, nil
	}

	line, offset := nextLine(content, 0)
	if strings.TrimRight(line, "\r") != "---" {
		return "", content, false, nil
	}

	lines := make([]string, 0, 8)
	for offset < len(content) {
		line, next := nextLine(content, offset)
		offset = next

		if strings.TrimRight(line, "\r") == "---" {
			return strings.Join(lines, "\n"), content[offset:], true, nil
		}

		lines = append(lines, strings.TrimRight(line, "\r"))
	}

	return "", nil, false, errors.New("frontmatter is missing a closing delimiter")
}

func ParseFrontmatter(block string) (*Frontmatter, error) {
	meta := &Frontmatter{}
	var (
		titleOK   bool
		tagsOK    bool
		sourcesOK bool
		updatedOK bool
	)

	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return nil, fmt.Errorf("invalid frontmatter line %q", line)
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		switch key {
		case "title":
			meta.Title = trimQuotes(value)
			titleOK = meta.Title != ""
		case "tags":
			items, err := parseListValue(value)
			if err != nil {
				return nil, fmt.Errorf("parse tags: %w", err)
			}
			meta.Tags = items
			tagsOK = true
		case "sources":
			items, err := parseListValue(value)
			if err != nil {
				return nil, fmt.Errorf("parse sources: %w", err)
			}
			meta.Sources = items
			sourcesOK = true
		case "updated":
			updated, err := time.Parse(dateLayout, trimQuotes(value))
			if err != nil {
				return nil, fmt.Errorf("parse updated: %w", err)
			}
			meta.Updated = updated
			updatedOK = true
		}
	}

	if !titleOK || !tagsOK || !sourcesOK || !updatedOK {
		return nil, errors.New("frontmatter requires title, tags, sources, and updated")
	}

	return meta, nil
}

func ParseRequiredFrontmatter(content []byte) (*Frontmatter, []byte, error) {
	block, body, has, err := SplitFrontmatter(content)
	if err != nil {
		return nil, nil, err
	}
	if !has {
		return nil, nil, errors.New("missing frontmatter")
	}

	meta, err := ParseFrontmatter(block)
	if err != nil {
		return nil, nil, err
	}

	return meta, body, nil
}

func parseOptionalFrontmatter(content []byte) (*Frontmatter, []byte, error) {
	block, body, has, err := SplitFrontmatter(content)
	if err != nil {
		return nil, nil, err
	}
	if !has {
		return nil, content, nil
	}

	meta, err := ParseFrontmatter(block)
	if err != nil {
		return nil, nil, err
	}

	return meta, body, nil
}

func parseListValue(value string) ([]string, error) {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "[") || !strings.HasSuffix(value, "]") {
		return nil, fmt.Errorf("expected list syntax, got %q", value)
	}

	inner := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(value, "["), "]"))
	if inner == "" {
		return []string{}, nil
	}

	parts := strings.Split(inner, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		item := trimQuotes(strings.TrimSpace(part))
		if item == "" {
			return nil, fmt.Errorf("empty list item in %q", value)
		}
		items = append(items, item)
	}

	return items, nil
}

func trimQuotes(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1]
		}
	}
	return value
}

func nextLine(content []byte, start int) (string, int) {
	for i := start; i < len(content); i++ {
		if content[i] == '\n' {
			return string(content[start:i]), i + 1
		}
	}
	return string(content[start:]), len(content)
}
