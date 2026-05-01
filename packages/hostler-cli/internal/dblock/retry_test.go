// trac: HAR-CM003
package dblock

import (
	"errors"
	"fmt"
	"testing"
)

// TestIsBusy_FallbackMessage — fallback that matches driver-wrapped messages.
func TestIsBusy_FallbackMessage(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{errors.New("database is locked (5)"), true},
		{errors.New("SQLITE_BUSY: ..."), true},
		{errors.New("database table is locked: tasks"), true},
		{errors.New("no such column"), false},
		{errors.New("syntax error"), false},
		{nil, false},
	}
	for i, c := range cases {
		got := IsBusy(c.err)
		if got != c.want {
			t.Errorf("[%d] err=%v got=%v want=%v", i, c.err, got, c.want)
		}
	}
}

// TestIsDisabled_OptOut — env-var inspection.
func TestIsDisabled_OptOut(t *testing.T) {
	cases := []struct {
		env  string
		want bool
	}{
		{"off", true},
		{"OFF", true},
		{"Off", true},
		{"on", false},
		{"", false},
		{"true", false},
	}
	for _, c := range cases {
		t.Setenv(envKey, c.env)
		got := IsDisabled()
		if got != c.want {
			t.Errorf("env=%q got=%v want=%v", c.env, got, c.want)
		}
	}
}

// TestIsBusy_NonBusyError — non-busy errors must fail immediately.
func TestIsBusy_NonBusyError(t *testing.T) {
	err := fmt.Errorf("some other error: %w", errors.New("constraint violation"))
	if IsBusy(err) {
		t.Errorf("constraint error was incorrectly classified as busy")
	}
}

// TestEnvKeyConstant — env key SSOT.
func TestEnvKeyConstant(t *testing.T) {
	if envKey != "HSTL_DB_LOCK_RETRY" {
		t.Errorf("env key SSOT drift: %q", envKey)
	}
	t.Setenv(envKey, "")
	if IsDisabled() {
		t.Errorf("retries must remain enabled when the env var is empty")
	}
}

// TestRetryMaxConstant — retry-max consistency.
func TestRetryMaxConstant(t *testing.T) {
	if retryMax != 3 {
		t.Errorf("retryMax SSOT drift: %d (expected 3)", retryMax)
	}
	if len(retryDelays) != retryMax {
		t.Errorf("retryDelays len=%d != retryMax=%d", len(retryDelays), retryMax)
	}
}
