package sprint

import "testing"

func TestAnalyzeScope_Empty(t *testing.T) {
	got := AnalyzeScope("Stabilization — strengthen the Result-section parser and tidy categories")
	if len(got) != 0 {
		t.Fatalf("expected 0 signals, got %d: %+v", len(got), got)
	}
}

func TestAnalyzeScope_ScopeBoundary(t *testing.T) {
	got := AnalyzeScope("integrate with other system + adopt traefik + external api integration")
	if len(got) < 3 {
		t.Fatalf("expected >= 3 signals, got %d: %+v", len(got), got)
	}
	categories := map[string]int{}
	for _, s := range got {
		categories[s.Category]++
	}
	if categories["scope_boundary"] < 1 {
		t.Errorf("expected scope_boundary >= 1, got %d", categories["scope_boundary"])
	}
	if categories["external_integration"] < 2 {
		t.Errorf("expected external_integration >= 2 (traefik + external api), got %d", categories["external_integration"])
	}
}

func TestAnalyzeScope_CaseInsensitive(t *testing.T) {
	got := AnalyzeScope("TRAEFIK integration + EXTERNAL API")
	if len(got) < 2 {
		t.Fatalf("case-insensitive matching failed: %+v", got)
	}
}

func TestScopeReviewBlocked_EmptySignals(t *testing.T) {
	if got := ScopeReviewBlocked(nil); got != "" {
		t.Fatalf("empty signal list should return empty string, got %q", got)
	}
}

func TestScopeReviewBlocked_MessageContainsHint(t *testing.T) {
	signals := AnalyzeScope("traefik integration")
	msg := ScopeReviewBlocked(signals)
	if msg == "" {
		t.Fatal("expected non-empty message when signals are present")
	}
	substrings := []string{"scope", "traefik", "--force"}
	for _, s := range substrings {
		if !containsLowerCase(msg, s) {
			t.Errorf("message missing substring %q: %s", s, msg)
		}
	}
}

func containsLowerCase(s, substr string) bool {
	return len(s) >= len(substr) && (len(substr) == 0 ||
		stringContainsFold(s, substr))
}

func stringContainsFold(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	for i := 0; i+len(substr) <= len(s); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			a, b := s[i+j], substr[j]
			if toLowerASCII(a) != toLowerASCII(b) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func toLowerASCII(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}
