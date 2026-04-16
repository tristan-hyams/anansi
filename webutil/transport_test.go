package webutil

import (
	"net/http"
	"testing"
	"time"
)

func TestTransport_ReturnsNonNil(t *testing.T) {
	tr := Transport()
	if tr == nil {
		t.Fatal("Transport() returned nil")
	}
}

func TestTransport_Singleton(t *testing.T) {
	a := Transport()
	b := Transport()
	if a != b {
		t.Fatal("Transport() returned different pointers")
	}
}

func TestNewClient_WrapsSharedTransport(t *testing.T) {
	client := NewClient(5 * time.Second)
	if client == nil {
		t.Fatal("NewClient returned nil")
	}

	tr, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("client transport is not *http.Transport")
	}
	if tr != Transport() {
		t.Fatal("client transport is not the shared singleton")
	}
}

func TestNewClient_SetsTimeout(t *testing.T) {
	timeout := 7 * time.Second
	client := NewClient(timeout)
	if client.Timeout != timeout {
		t.Fatalf("expected timeout %v, got %v", timeout, client.Timeout)
	}
}
