package server

import (
	"embed"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"

	"wikiclaudia/wiki"
)

//go:embed templates/*.html static/style.css
var embeddedFiles embed.FS

type App struct {
	site      *wiki.Site
	renderer  *Renderer
	templates *template.Template
	styleCSS  []byte
	mux       *http.ServeMux
}

type pageData struct {
	SiteTitle      string
	DocumentTitle  string
	ActiveHome     bool
	ActiveOverview bool
	Meta           *metaData
	TOC            []TOCEntry
	HTML           template.HTML
}

type metaData struct {
	Updated string
	Tags    []string
	Sources []metaLink
}

type metaLink struct {
	Label   string
	Href    string
	Missing bool
}

func New(site *wiki.Site) (*App, error) {
	templates, err := template.ParseFS(embeddedFiles, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	styleCSS, err := embeddedFiles.ReadFile("static/style.css")
	if err != nil {
		return nil, fmt.Errorf("read embedded stylesheet: %w", err)
	}

	app := &App{
		site:      site,
		renderer:  NewRenderer(),
		templates: templates,
		styleCSS:  styleCSS,
	}

	mux := http.NewServeMux()
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir(filepath.Join(site.Root, "assets")))))
	mux.HandleFunc("/static/style.css", app.handleStyle)
	mux.HandleFunc("/overview", app.handleOverview)
	mux.HandleFunc("/wiki/", app.handlePage)
	mux.HandleFunc("/", app.handleIndex)
	app.mux = mux

	return app, nil
}

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	a.mux.ServeHTTP(w, r)
}

func (a *App) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	article, err := wiki.LoadIndex(a.site.Root)
	if err != nil {
		http.Error(w, fmt.Sprintf("load home page: %v", err), http.StatusInternalServerError)
		return
	}

	a.renderArticle(w, article, true, false)
}

func (a *App) handleOverview(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/overview" {
		http.NotFound(w, r)
		return
	}

	article, err := wiki.LoadOverview(a.site.Root)
	if err != nil {
		http.Error(w, fmt.Sprintf("load overview: %v", err), http.StatusInternalServerError)
		return
	}

	a.renderArticle(w, article, false, true)
}

func (a *App) handlePage(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/wiki/")
	if slug == "" || strings.Contains(slug, "/") {
		http.NotFound(w, r)
		return
	}

	article, err := wiki.LoadPage(a.site.Root, slug)
	if err != nil {
		if errors.Is(err, wiki.ErrPageNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, fmt.Sprintf("load page: %v", err), http.StatusInternalServerError)
		return
	}

	a.renderArticle(w, article, false, false)
}

func (a *App) handleStyle(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/static/style.css" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	_, _ = w.Write(a.styleCSS)
}

func (a *App) renderArticle(w http.ResponseWriter, article *wiki.Article, activeHome bool, activeOverview bool) {
	linkedSlugs := collectWikiLinkSlugs(article.Body)
	if article.Meta != nil {
		linkedSlugs = append(linkedSlugs, article.Meta.Sources...)
	}
	targets := wiki.ResolveLinks(a.site.Root, linkedSlugs)

	bodyHTML, toc, err := a.renderer.Render(article.Body, targets)
	if err != nil {
		http.Error(w, fmt.Sprintf("render article: %v", err), http.StatusInternalServerError)
		return
	}

	data := pageData{
		SiteTitle:      a.site.Schema.Domain,
		DocumentTitle:  documentTitle(article, a.site.Schema.Domain),
		ActiveHome:     activeHome,
		ActiveOverview: activeOverview,
		TOC:            toc,
		HTML:           bodyHTML,
	}

	if article.Meta != nil {
		data.Meta = &metaData{
			Updated: article.Meta.UpdatedString(),
			Tags:    article.Meta.Tags,
			Sources: buildMetaLinks(article.Meta.Sources, targets),
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.templates.ExecuteTemplate(w, "page", data); err != nil {
		http.Error(w, fmt.Sprintf("render template: %v", err), http.StatusInternalServerError)
	}
}

func buildMetaLinks(slugs []string, targets map[string]wiki.LinkTarget) []metaLink {
	links := make([]metaLink, 0, len(slugs))
	for _, slug := range slugs {
		target, ok := targets[slug]
		if !ok {
			target = wiki.LinkTarget{
				Slug:  slug,
				Title: wiki.HumanizeSlug(slug),
			}
		}

		label := target.Title
		if label == "" {
			label = wiki.HumanizeSlug(slug)
		}

		links = append(links, metaLink{
			Label:   label,
			Href:    "/wiki/" + slug,
			Missing: !target.Exists,
		})
	}
	return links
}

func documentTitle(article *wiki.Article, siteTitle string) string {
	if article.Meta != nil && article.Meta.Title != "" {
		return article.Meta.Title + " | " + siteTitle
	}
	return "Home | " + siteTitle
}
