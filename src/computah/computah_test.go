package computah

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestExtractPrompt(t *testing.T) {
	tests := []struct {
		input  string
		want   string
		wantOk bool
	}{
		{"computah summarize dune", "summarize dune", true},
		{"Computah, find me the youtube video for northernlion", "find me the youtube video for northernlion", true},
		{"COMPUTAH: what is go?", "what is go?", true},
		{"computah", "", false},
		{"computah   ", "", false},
		{"computah, ", "", false},
		{"computaholic test", "", false},
		{"hello computah test", "", false},
	}

	for _, tc := range tests {
		got, ok := extractPrompt(tc.input)
		if ok != tc.wantOk || got != tc.want {
			t.Errorf("extractPrompt(%q) = (%q, %v), want (%q, %v)", tc.input, got, ok, tc.want, tc.wantOk)
		}
	}
}

func TestRateLimiterRollingWindow(t *testing.T) {
	rl := &rateLimiter{
		userSubmissions: make(map[string][]time.Time),
	}

	base := time.Now()
	user := "user123"

	// 1. Test 5-per-5m burst limit
	for i := 0; i < 5; i++ {
		_, allowed := rl.allow(user, base.Add(time.Duration(i)*time.Minute))
		if !allowed {
			t.Fatalf("expected request %d to be allowed", i+1)
		}
	}

	reason, allowed := rl.allow(user, base.Add(4*time.Minute))
	if allowed {
		t.Fatalf("expected 6th request inside 5-minute window to be rejected")
	}
	if !strings.Contains(reason, "per 5 minutes") {
		t.Fatalf("expected burst limit reason, got %q", reason)
	}

	// 2. Test 20-per-24h user limit (spaced 10 minutes apart to avoid 5m burst limit)
	for i := 5; i < 20; i++ {
		_, allowed := rl.allow(user, base.Add(time.Duration(i*10)*time.Minute))
		if !allowed {
			t.Fatalf("expected request %d to be allowed", i+1)
		}
	}

	reason, allowed = rl.allow(user, base.Add(210*time.Minute))
	if allowed {
		t.Fatalf("expected 21st request for same user within 24h to be rejected")
	}
	if !strings.Contains(reason, "Daily user limit") {
		t.Fatalf("expected daily user limit reason, got %q", reason)
	}

	// 3. Test 100-per-24h server-wide limit across multiple users
	for u := 0; u < 4; u++ {
		otherUser := fmt.Sprintf("otherUser-%d", u)
		for i := 0; i < 20; i++ {
			_, allowed := rl.allow(otherUser, base.Add(time.Duration(i*10)*time.Minute))
			if !allowed {
				t.Fatalf("expected user %s request %d to be allowed", otherUser, i+1)
			}
		}
	}

	// Server now has 20 + (4 * 20) = 100 requests in the last 24h
	reason, allowed = rl.allow("freshUser", base.Add(300*time.Minute))
	if allowed {
		t.Fatalf("expected 101st server request within 24h to be rejected")
	}
	if !strings.Contains(reason, "Daily server limit") {
		t.Fatalf("expected daily server limit reason, got %q", reason)
	}
}
