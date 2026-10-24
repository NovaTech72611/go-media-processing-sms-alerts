package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendUsesContractBodyField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if payload["to"] != "+15550001" || payload["body"] != "ready" || len(payload) != 2 {
			t.Errorf("unexpected SMS payload: %v", payload)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"data":{"message_id":"test-id"}}`))
	}))
	defer server.Close()
	client := &SMSClient{BaseURL: server.URL, Key: "test", HTTP: server.Client()}
	if id, err := client.Send(context.Background(), "+15550001", "ready", "test-request"); err != nil || id != "test-id" {
		t.Fatalf("Send() = %q, %v", id, err)
	}
}

func TestShouldNotifyOnlyProcessedAssets(t *testing.T) {
	cases := []struct {
		name  string
		asset Asset
		want  bool
	}{
		{"processed creator", Asset{ID: "a1", CreatorPhone: "+15550001", Processed: true}, true},
		{"pending job", Asset{ID: "a2", CreatorPhone: "+15550002", Processed: false}, false},
		{"no destination", Asset{ID: "a3", Processed: true}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldNotify(tc.asset); got != tc.want {
				t.Fatalf("shouldNotify() = %v, want %v", got, tc.want)
			}
		})
	}
}
