package aibot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestQRGenerateURL(t *testing.T) {
	u := QRGenerateURL("wecom-cli", 1)
	if !strings.Contains(u, "source=wecom-cli") || !strings.Contains(u, "plat=1") {
		t.Fatalf("unexpected url: %s", u)
	}
}

func TestQRGenPageURL(t *testing.T) {
	u := QRGenPageURL("abc123")
	if !strings.Contains(u, "scode=abc123") {
		t.Fatalf("unexpected url: %s", u)
	}
}

func TestQRPlatformFromGOOS(t *testing.T) {
	p := QRPlatformFromGOOS()
	if p < 0 || p > 3 {
		t.Fatalf("plat out of range: %d", p)
	}
}

func TestFetchQRCodeSession(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]string{
				"scode":    "s1",
				"auth_url": "https://example.com/auth",
			},
		})
	}))
	defer ts.Close()

	sess, err := FetchQRCodeSession(context.Background(), QRScanHTTPConfig{
		GenerateRequestURL: ts.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if sess.Scode != "s1" || sess.AuthURL != "https://example.com/auth" {
		t.Fatalf("session: %+v", sess)
	}
}

func TestWaitForScanResult(t *testing.T) {
	var n atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if n.Add(1) < 2 {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]string{"status": "pending"},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"status": "success",
				"bot_info": map[string]string{
					"botid":  "bid",
					"secret": "sec",
				},
			},
		})
	}))
	defer ts.Close()

	cfg := QRScanHTTPConfig{
		QueryURLBase: ts.URL,
		PollInterval: 10 * time.Millisecond,
		PollTimeout:  2 * time.Second,
	}
	creds, err := WaitForScanResult(context.Background(), cfg, "scode-x")
	if err != nil {
		t.Fatal(err)
	}
	if creds.BotID != "bid" || creds.Secret != "sec" {
		t.Fatalf("creds: %+v", creds)
	}
}

func TestWaitForScanResult_Timeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]string{"status": "waiting"},
		})
	}))
	defer ts.Close()

	cfg := QRScanHTTPConfig{
		QueryURLBase: ts.URL,
		PollInterval: 20 * time.Millisecond,
		PollTimeout:  50 * time.Millisecond,
	}
	_, err := WaitForScanResult(context.Background(), cfg, "scode-x")
	if err != ErrQRScanTimeout {
		t.Fatalf("want ErrQRScanTimeout, got %v", err)
	}
}
