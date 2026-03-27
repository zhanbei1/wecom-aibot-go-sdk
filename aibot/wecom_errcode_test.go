package aibot

import (
	"strings"
	"testing"
)

func TestAuthFailureUserHint_600041_invalidSecret(t *testing.T) {
	h := AuthFailureUserHint(600041, "invalid secret")
	if h == "" {
		t.Fatal("expected non-empty hint")
	}
	for _, part := range []string{"invalid secret", "全局错误码", "分页"} {
		if !strings.Contains(h, part) {
			t.Fatalf("hint should contain %q, got %q", part, h)
		}
	}
}

func TestAuthFailureUserHint_secretGeneric(t *testing.T) {
	h := AuthFailureUserHint(99999, "bad secret")
	if h == "" || !strings.Contains(h, "bot_id") {
		t.Fatalf("unexpected hint: %q", h)
	}
}

func TestAuthFailureUserHint_empty(t *testing.T) {
	if AuthFailureUserHint(0, "") != "" {
		t.Fatal("expected empty")
	}
}
