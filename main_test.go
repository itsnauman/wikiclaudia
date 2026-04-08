package main

import (
	"net"
	"strconv"
	"strings"
	"testing"
)

func TestListenReturnsHelpfulErrorWhenPortBusy(t *testing.T) {
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("open busy listener: %v", err)
	}
	t.Cleanup(func() {
		_ = busy.Close()
	})

	port := busy.Addr().(*net.TCPAddr).Port
	_, err = listen("127.0.0.1", port)
	if err == nil {
		t.Fatal("expected listen error")
	}

	if !strings.Contains(err.Error(), "already in use") && !strings.Contains(err.Error(), strconv.Itoa(port)) {
		t.Fatalf("unexpected error: %v", err)
	}
}
