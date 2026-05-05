package toxicity

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"dxcluster/spot"
)

func TestClientSendsOnlyCommentAndParsesToxic(t *testing.T) {
	t.Setenv("DXC_TEST_TOXICITY_TOKEN", "secret")
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Fatalf("unexpected auth header %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{"status":"toxic","categories":["S10"],"model":"mock"}`))
	}))
	defer server.Close()

	client := NewClient(Config{Endpoint: server.URL, BearerTokenEnv: "DXC_TEST_TOXICITY_TOKEN"}, server.Client())
	decision, err := client.Classify(context.Background(), "merci pour le contact")
	if err != nil {
		t.Fatalf("Classify failed: %v", err)
	}
	if decision.Status != spot.ToxicityToxic || decision.Model != "mock" || len(decision.Categories) != 1 {
		t.Fatalf("unexpected decision %#v", decision)
	}
	if len(received) != 1 || received["comment"] != "merci pour le contact" {
		t.Fatalf("expected only comment payload, got %#v", received)
	}
}

func TestClientFailsOnHTTPAndUnknownStatus(t *testing.T) {
	t.Setenv("DXC_TEST_TOXICITY_TOKEN", "secret")
	for name, response := range map[string]struct {
		status int
		body   string
	}{
		"rate-limit": {status: http.StatusTooManyRequests, body: `{"error":"rate_limited"}`},
		"server":     {status: http.StatusInternalServerError, body: `{"error":"boom"}`},
		"unknown":    {status: http.StatusOK, body: `{"status":"maybe"}`},
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(response.status)
				_, _ = w.Write([]byte(response.body))
			}))
			defer server.Close()

			client := NewClient(Config{Endpoint: server.URL, BearerTokenEnv: "DXC_TEST_TOXICITY_TOKEN"}, server.Client())
			if _, err := client.Classify(context.Background(), "bad comment"); err == nil {
				t.Fatalf("expected error for %s", name)
			}
		})
	}
}

func TestClientRequiresConfiguredToken(t *testing.T) {
	_ = os.Unsetenv("DXC_TEST_TOXICITY_TOKEN")
	client := NewClient(Config{Endpoint: "https://example.invalid/classify", BearerTokenEnv: "DXC_TEST_TOXICITY_TOKEN"}, nil)
	if _, err := client.Classify(context.Background(), "comment"); err == nil {
		t.Fatal("expected missing token error")
	}
}
