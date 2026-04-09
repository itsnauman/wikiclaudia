package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/itsnauman/wikiclaudia/server"
	"github.com/itsnauman/wikiclaudia/wiki"
)

type config struct {
	host string
	port int
}

func main() {
	cfg := config{
		host: "127.0.0.1",
		port: 8080,
	}

	flag.StringVar(&cfg.host, "host", cfg.host, "host to bind")
	flag.IntVar(&cfg.port, "port", cfg.port, "port to bind")
	flag.Parse()

	if err := run(os.Stdout, os.Stderr, cfg); err != nil {
		errStyle := newStyle(os.Stderr)
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "  "+errStyle.errorLine(err.Error()))
		fmt.Fprintln(os.Stderr)
		os.Exit(1)
	}
}

func run(stdout io.Writer, stderr io.Writer, cfg config) error {
	out := newStyle(stdout)
	errs := newStyle(stderr)

	if cfg.port <= 0 || cfg.port > 65535 {
		return fmt.Errorf("invalid port %d", cfg.port)
	}

	root, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("resolve working directory: %w", err)
	}

	site, err := wiki.ValidateRoot(root)
	if err != nil {
		return err
	}

	app, err := server.New(site)
	if err != nil {
		return fmt.Errorf("build server: %w", err)
	}

	listener, err := listen(cfg.host, cfg.port)
	if err != nil {
		return err
	}

	serveHost := browserHost(cfg.host)
	serveURL := fmt.Sprintf("http://%s", net.JoinHostPort(serveHost, strconv.Itoa(cfg.port)))

	printBanner(stdout, out, site, countPages(site.Root))
	printReady(stdout, out, serveURL)

	if err := openBrowser(serveURL); err != nil {
		fmt.Fprintln(stderr, "  "+errs.warning("! could not open browser automatically"))
		fmt.Fprintln(stderr)
	}

	printHint(stdout, out)

	httpServer := &http.Server{Handler: app}

	shutdownCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serveErr := make(chan error, 1)
	go func() {
		err := httpServer.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- fmt.Errorf("serve %s: %w", serveURL, err)
			return
		}
		serveErr <- nil
	}()

	select {
	case err := <-serveErr:
		return err
	case <-shutdownCtx.Done():
		stop()
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "  "+out.dim("○ stopping…"))

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(ctx)

		if err := <-serveErr; err != nil {
			return err
		}

		fmt.Fprintln(stdout, "  "+out.dim("○ stopped. bye."))
		fmt.Fprintln(stdout)
		return nil
	}
}

func printBanner(w io.Writer, s style, site *wiki.Site, pageCount int) {
	label := func(text string) string {
		return s.dim(fmt.Sprintf("  %-8s", text))
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "  "+s.bold("wikiclaudia"))
	fmt.Fprintln(w)
	fmt.Fprintln(w, label("domain")+site.Schema.Domain)
	fmt.Fprintln(w, label("pages")+strconv.Itoa(pageCount))
	fmt.Fprintln(w, label("root")+shortenPath(site.Root))
	fmt.Fprintln(w)
}

func printReady(w io.Writer, s style, serveURL string) {
	fmt.Fprintln(w, "  "+s.ready("●")+"  ready  "+s.link(serveURL))
	fmt.Fprintln(w)
}

func printHint(w io.Writer, s style) {
	fmt.Fprintln(w, "  "+s.dim("press ctrl+c to stop"))
	fmt.Fprintln(w)
}

func shortenPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if path == home {
		return "~"
	}
	if strings.HasPrefix(path, home+string(os.PathSeparator)) {
		return "~" + strings.TrimPrefix(path, home)
	}
	return path
}

func countPages(root string) int {
	entries, err := os.ReadDir(filepath.Join(root, "wiki", "pages"))
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".md" {
			count++
		}
	}
	return count
}

func listen(host string, port int) (net.Listener, error) {
	address := net.JoinHostPort(host, strconv.Itoa(port))
	listener, err := net.Listen("tcp", address)
	if err == nil {
		return listener, nil
	}

	if errors.Is(err, syscall.EADDRINUSE) {
		return nil, fmt.Errorf("port %d is already in use", port)
	}

	return nil, fmt.Errorf("listen on %s: %w", address, err)
}

func browserHost(host string) string {
	switch host {
	case "", "0.0.0.0", "::":
		return "127.0.0.1"
	default:
		return host
	}
}

func openBrowser(target string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", target)
	case "linux":
		cmd = exec.Command("xdg-open", target)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		return fmt.Errorf("unsupported platform %s", runtime.GOOS)
	}

	return cmd.Start()
}
