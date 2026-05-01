package envalias

import (
	"os"
	"testing"
)

// envalias unit tests — Lookup / LookupOK against HSTL_<key>.

// Lookup returns the value of HSTL_<key>.
func TestLookup_Set(t *testing.T) {
	t.Setenv("HSTL_FOO", "hstl-value")
	if got := Lookup("FOO"); got != "hstl-value" {
		t.Errorf("got=%q want=hstl-value", got)
	}
}

// Lookup returns "" when HSTL_<key> is unset.
func TestLookup_Unset(t *testing.T) {
	if err := unsetEnv(t, "HSTL_QUX"); err != nil {
		t.Fatal(err)
	}
	if got := Lookup("QUX"); got != "" {
		t.Errorf("got=%q want empty", got)
	}
}

// Lookup returns "" when HSTL_<key> is set to the empty string.
func TestLookup_Empty(t *testing.T) {
	t.Setenv("HSTL_HUM", "")
	if got := Lookup("HUM"); got != "" {
		t.Errorf("got=%q want empty", got)
	}
}

// LookupOK reports the found flag correctly.
func TestLookupOK(t *testing.T) {
	t.Setenv("HSTL_OK_TEST", "v1")
	if v, ok := LookupOK("OK_TEST"); !ok || v != "v1" {
		t.Errorf("got=(%q,%v) want=(v1,true)", v, ok)
	}

	if err := unsetEnv(t, "HSTL_NOT_SET"); err != nil {
		t.Fatal(err)
	}
	if _, ok := LookupOK("NOT_SET"); ok {
		t.Errorf("unexpected ok=true")
	}
}

// helper — t.Setenv does not unset directly, so use os.Unsetenv + Cleanup.
func unsetEnv(t *testing.T, key string) error {
	t.Helper()
	prev, had := os.LookupEnv(key)
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(key, prev)
		} else {
			_ = os.Unsetenv(key)
		}
	})
	return os.Unsetenv(key)
}
