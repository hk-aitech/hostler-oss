// Unit tests for normalizeType.
package cmd

import "testing"

func TestNormalizeType(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		// canonical inputs
		{"feature", "feature", false},
		{"bugfix", "bugfix", false},
		{"docs", "docs", false},
		{"refactor", "refactor", false},
		{"infra", "infra", false},
		{"test", "test", false},
		{"chore", "chore", false},
		{"spike", "spike", false},
		{"hotfix", "hotfix", false},

		// case normalization
		{"Feature", "feature", false},
		{"INFRA", "infra", false},

		// whitespace / trim
		{"  bugfix  ", "bugfix", false},

		// alias mapping
		{"fix", "bugfix", false}, // observed drift
		{"bug", "bugfix", false},
		{"bug-fix", "bugfix", false},
		{"bug fix", "bugfix", false}, // space -> hyphen, then alias
		{"doc", "docs", false},
		{"documentation", "docs", false},
		{"feat", "feature", false},

		// empty input -> pass (optional flag)
		{"", "", false},
		{"   ", "", false},

		// rejected cases
		{"support", "", true},
		{"p0", "", true},
		{"new-feature", "", true},
	}

	for _, c := range cases {
		got, err := normalizeType(c.in)
		if (err != nil) != c.wantErr {
			t.Errorf("normalizeType(%q) err=%v, wantErr=%v", c.in, err, c.wantErr)
			continue
		}
		if err == nil && got != c.want {
			t.Errorf("normalizeType(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// idempotence: feeding canonical values back in must return the same value.
func TestNormalizeType_Idempotence(t *testing.T) {
	canonicals := []string{"feature", "bugfix", "docs", "refactor", "infra", "test", "chore", "spike", "hotfix"}
	for _, c := range canonicals {
		got, err := normalizeType(c)
		if err != nil || got != c {
			t.Errorf("idempotence failed: normalizeType(%q) = (%q, %v)", c, got, err)
		}
	}
}
