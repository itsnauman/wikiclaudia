package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"syscall"

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
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(stdout io.Writer, stderr io.Writer, cfg config) error {
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
	fmt.Fprintf(stdout, "%s\n", serveURL)

	logger := log.New(stderr, "", 0)
	if err := openBrowser(serveURL); err != nil {
		logger.Printf("warning: failed to open browser: %v", err)
	}

	httpServer := &http.Server{
		Handler: app,
	}

	if err := httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve %s: %w", serveURL, err)
	}

	return nil
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
