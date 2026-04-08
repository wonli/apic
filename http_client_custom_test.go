package apic

import (
	"net/http"
	"testing"
)

func TestApiClientsWithHTTPClient(t *testing.T) {
	custom := &http.Client{}
	client := NewApiClient().WithHTTPClient(custom)
	if got := client.HTTPClient(); got != custom {
		t.Fatalf("expected custom http client, got %#v", got)
	}
}

func TestApiClientsHTTPClientFallsBackToDefault(t *testing.T) {
	client := NewApiClient()
	if got := client.HTTPClient(); got == nil {
		t.Fatal("expected default http client")
	}
}
