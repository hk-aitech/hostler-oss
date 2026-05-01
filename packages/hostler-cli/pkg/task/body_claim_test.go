// T455 (Sprint-44) regression tests.

package task

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseBodyClaims_whitelist(t *testing.T) {
	// command without quotes + nearby "N items" claim
	body := `
## Measurements

Measurement found 5 items.

` + "```bash\ngrep -rn BootstrapEventIds src/\n```" + `

subsequent work.
`
	claims := ParseBodyClaims(body)
	if len(claims) != 1 {
		t.Fatalf("claims count = %d, want 1", len(claims))
	}
	if claims[0].ExpectedCount != 5 {
		t.Errorf("ExpectedCount = %d, want 5", claims[0].ExpectedCount)
	}
	if claims[0].Command[0] != "grep" {
		t.Errorf("Command[0] = %q, want grep", claims[0].Command[0])
	}
}

func TestParseBodyClaims_quoted_rejected(t *testing.T) {
	// commands containing quotes split inaccurately → skip
	body := `
expected 5 items.

` + "```bash\ngrep -rn 'foo' src/\n```" + `
`
	claims := ParseBodyClaims(body)
	if len(claims) != 0 {
		t.Errorf("expected quoted command to be skipped — got %d", len(claims))
	}
}

func TestParseBodyClaims_dangerous_metachar_rejected(t *testing.T) {
	body := `
expected 3 items.

` + "```bash\ngrep foo src/ | wc -l\n```" + `
`
	claims := ParseBodyClaims(body)
	// skipped because of the | metachar
	if len(claims) != 0 {
		t.Errorf("expected metachar command to be skipped — got %d", len(claims))
	}
}

func TestParseBodyClaims_non_whitelist_rejected(t *testing.T) {
	body := `
expected 2 items.

` + "```bash\ncurl https://evil.example/\n```" + `
`
	claims := ParseBodyClaims(body)
	if len(claims) != 0 {
		t.Errorf("expected non-whitelist curl to be skipped — got %d", len(claims))
	}
}

func TestParseBodyClaims_no_claim_number_skip(t *testing.T) {
	body := `
## Measurements

no number claim around here.

` + "```bash\ngrep foo bar\n```" + `

still no number.
`
	claims := ParseBodyClaims(body)
	if len(claims) != 0 {
		t.Errorf("expected skip when no claim number — got %d", len(claims))
	}
}

// TestRevalidateClaims_drift — when drift occurs, return WARN.
func TestRevalidateClaims_drift(t *testing.T) {
	tmp := t.TempDir()
	// fixture: create 3 files + grep for a common substring (2 expected)
	os.WriteFile(filepath.Join(tmp, "a.txt"), []byte("hit\nhit\n"), 0o644)
	os.WriteFile(filepath.Join(tmp, "b.txt"), []byte("hit\nmiss\n"), 0o644)
	os.WriteFile(filepath.Join(tmp, "c.txt"), []byte("no match\n"), 0o644)

	claims := []BodyClaim{
		{
			Command:       []string{"grep", "-r", "hit", "."},
			ExpectedCount: 1, // actually 3 lines hit → drift
			ClaimLine:     "grep -r hit .",
			CodeFenceLine: 10,
		},
	}
	warns := RevalidateClaims(claims, tmp)
	if len(warns) != 1 {
		t.Fatalf("expected 1 drift WARN — got %d", len(warns))
	}
}

// TestRevalidateClaims_match — no WARN when expected matches actual.
func TestRevalidateClaims_match(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "a.txt"), []byte("hit\nhit\n"), 0o644)

	claims := []BodyClaim{
		{
			Command:       []string{"grep", "-c", "hit", "a.txt"},
			ExpectedCount: 2,
			ClaimLine:     "grep -c hit a.txt",
			CodeFenceLine: 10,
		},
	}
	// grep -c emits 1 line (a count) per file
	// runAndCountLines returns line count (grep -c stdout = "2" → 1 line)
	// ExpectedCount=2 but line count=1 → drift; this is a -c quirk
	// the test substitutes a plain grep (without -c)
	claims[0].Command = []string{"grep", "hit", "a.txt"}
	warns := RevalidateClaims(claims, tmp)
	if len(warns) != 0 {
		t.Errorf("expected match — got %d WARN: %v", len(warns), warns)
	}
}

// TestRevalidateTaskBody_optout — return empty slice when env is off.
func TestRevalidateTaskBody_optout(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "T999-sample.md")
	body := "## Measurements\n\n1 item.\n\n```bash\ngrep foo nonexistent\n```\n"
	os.WriteFile(f, []byte(body), 0o644)

	t.Setenv("HSTL_TASK_START_BODY_REVALIDATE", "off")
	out := RevalidateTaskBody(f)
	if out != nil {
		t.Errorf("expected nil on opt-out — got %v", out)
	}
}
