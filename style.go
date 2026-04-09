package main

import (
	"io"
	"os"
)

const (
	ansiReset     = "\x1b[0m"
	ansiBold      = "\x1b[1m"
	ansiDim       = "\x1b[2m"
	ansiUnderline = "\x1b[4m"
	ansiRed       = "\x1b[31m"
	ansiGreen     = "\x1b[32m"
	ansiYellow    = "\x1b[33m"
	ansiCyan      = "\x1b[36m"
)

// style applies ANSI escape codes when the target writer is an interactive
// terminal. When color is disabled (piped output, NO_COLOR, non-file writer)
// every helper returns its input unchanged, so the output stays clean.
type style struct {
	color bool
}

func newStyle(w io.Writer) style {
	if os.Getenv("NO_COLOR") != "" {
		return style{}
	}
	f, ok := w.(*os.File)
	if !ok {
		return style{}
	}
	info, err := f.Stat()
	if err != nil {
		return style{}
	}
	if info.Mode()&os.ModeCharDevice == 0 {
		return style{}
	}
	return style{color: true}
}

func (s style) wrap(codes, text string) string {
	if !s.color {
		return text
	}
	return codes + text + ansiReset
}

func (s style) bold(text string) string    { return s.wrap(ansiBold, text) }
func (s style) dim(text string) string     { return s.wrap(ansiDim, text) }
func (s style) ready(text string) string   { return s.wrap(ansiGreen, text) }
func (s style) warning(text string) string { return s.wrap(ansiYellow, text) }

func (s style) link(text string) string {
	return s.wrap(ansiCyan+ansiUnderline, text)
}

func (s style) errorLine(text string) string {
	return s.wrap(ansiRed, "✗ ") + text
}
