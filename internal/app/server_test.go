package app

import (
	"net/http"
	"testing"
	"time"
)

func TestNewHTTPServerTimeouts(t *testing.T) {
	server := NewHTTPServer(":8080", http.NotFoundHandler())

	if server.ReadHeaderTimeout != 5*time.Second {
		t.Fatalf("ReadHeaderTimeout = %s, want 5s", server.ReadHeaderTimeout)
	}
	if server.ReadTimeout != 0 {
		t.Fatalf("ReadTimeout = %s, want disabled", server.ReadTimeout)
	}
	if server.WriteTimeout != 0 {
		t.Fatalf("WriteTimeout = %s, want disabled", server.WriteTimeout)
	}
	if server.IdleTimeout != time.Minute {
		t.Fatalf("IdleTimeout = %s, want 1m", server.IdleTimeout)
	}
}
