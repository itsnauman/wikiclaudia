package wiki_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/itsnauman/wikiclaudia/testfixture"
	"github.com/itsnauman/wikiclaudia/wiki"
)

func TestValidateRootAcceptsMinimalWiki(t *testing.T) {
	root := t.TempDir()
	if err := testfixture.WriteMinimalWiki(root, testfixture.Options{}); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	site, err := wiki.ValidateRoot(root)
	if err != nil {
		t.Fatalf("ValidateRoot returned error: %v", err)
	}

	if site.Schema.Domain != "Test Domain" {
		t.Fatalf("unexpected domain %q", site.Schema.Domain)
	}
}

func TestValidateRootRejectsMissingRequiredEntry(t *testing.T) {
	root := t.TempDir()
	if err := testfixture.WriteMinimalWiki(root, testfixture.Options{
		MissingRequired: filepath.Join("wiki", "overview.md"),
	}); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	_, err := wiki.ValidateRoot(root)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "missing required file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRootRejectsNestedPageDirectory(t *testing.T) {
	root := t.TempDir()
	if err := testfixture.WriteMinimalWiki(root, testfixture.Options{
		NestedPageDir: true,
	}); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	_, err := wiki.ValidateRoot(root)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "must be flat") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRootRejectsMalformedPageFrontmatter(t *testing.T) {
	root := t.TempDir()
	if err := testfixture.WriteMinimalWiki(root, testfixture.Options{
		InvalidPageFrontmatter: true,
	}); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	_, err := wiki.ValidateRoot(root)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "frontmatter") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRootRejectsSchemaPathMismatch(t *testing.T) {
	root := t.TempDir()
	if err := testfixture.WriteMinimalWiki(root, testfixture.Options{
		SchemaPath: "/tmp/not-the-root",
	}); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	_, err := wiki.ValidateRoot(root)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "does not match current directory") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadOverviewStripsFrontmatter(t *testing.T) {
	root := t.TempDir()
	if err := testfixture.WriteMinimalWiki(root, testfixture.Options{}); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	article, err := wiki.LoadOverview(root)
	if err != nil {
		t.Fatalf("LoadOverview returned error: %v", err)
	}

	if strings.Contains(article.Body, "title: Overview") {
		t.Fatalf("frontmatter was not stripped from article body: %q", article.Body)
	}
	if article.Meta == nil || article.Meta.Title != "Overview" {
		t.Fatalf("unexpected frontmatter: %#v", article.Meta)
	}
}
