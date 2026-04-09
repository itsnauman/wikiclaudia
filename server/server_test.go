package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/itsnauman/wikiclaudia/testfixture"
	"github.com/itsnauman/wikiclaudia/wiki"
)

func newTestApp(t *testing.T) (*App, string) {
	t.Helper()

	root := t.TempDir()
	if err := testfixture.WriteMinimalWiki(root, testfixture.Options{}); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	site, err := wiki.ValidateRoot(root)
	if err != nil {
		t.Fatalf("ValidateRoot returned error: %v", err)
	}

	app, err := New(site)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	return app, root
}

func TestRoutesServeExpectedContent(t *testing.T) {
	app, _ := newTestApp(t)

	server := httptest.NewServer(app)
	t.Cleanup(server.Close)

	assertBodyContains(t, server.URL+"/", http.StatusOK, "Home")
	assertBodyContains(t, server.URL+"/", http.StatusOK, "data-theme-toggle")
	assertBodyContains(t, server.URL+"/overview", http.StatusOK, "Overview")
	assertBodyContains(t, server.URL+"/wiki/alpha-page", http.StatusOK, "Alpha Page")
}

func TestMissingPageReturns404(t *testing.T) {
	app, _ := newTestApp(t)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/wiki/does-not-exist", nil)
	app.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
}

func TestAssetsRouteServesLocalFiles(t *testing.T) {
	app, _ := newTestApp(t)

	server := httptest.NewServer(app)
	t.Cleanup(server.Close)

	response, err := http.Get(server.URL + "/assets/diagram.txt")
	if err != nil {
		t.Fatalf("GET asset: %v", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.StatusCode)
	}
	if strings.TrimSpace(string(body)) != "asset-body" {
		t.Fatalf("unexpected asset body %q", string(body))
	}
}

func TestServerReadsFreshContentOnEachRequest(t *testing.T) {
	app, root := newTestApp(t)

	server := httptest.NewServer(app)
	t.Cleanup(server.Close)

	assertBodyContains(t, server.URL+"/wiki/alpha-page", http.StatusOK, "Original text")

	updatedPage := `---
title: Alpha Page
tags: [alpha, testing]
sources: [source-entry]
updated: 2026-04-08
---

# Alpha Page

Updated text after edit.
`
	pagePath := filepath.Join(root, "wiki", "pages", "alpha-page.md")
	if err := os.WriteFile(pagePath, []byte(updatedPage), 0o644); err != nil {
		t.Fatalf("write updated page: %v", err)
	}

	assertBodyContains(t, server.URL+"/wiki/alpha-page", http.StatusOK, "Updated text after edit")
}

func assertBodyContains(t *testing.T, targetURL string, wantStatus int, fragment string) {
	t.Helper()

	response, err := http.Get(targetURL)
	if err != nil {
		t.Fatalf("GET %s: %v", targetURL, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read body from %s: %v", targetURL, err)
	}

	if response.StatusCode != wantStatus {
		t.Fatalf("GET %s returned status %d, want %d", targetURL, response.StatusCode, wantStatus)
	}
	if !strings.Contains(string(body), fragment) {
		t.Fatalf("GET %s body missing %q: %s", targetURL, fragment, string(body))
	}
}
