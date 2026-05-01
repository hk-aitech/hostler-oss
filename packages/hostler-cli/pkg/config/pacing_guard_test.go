package config

import "testing"

// TestPacingGuard_DefaultTrue — defaults to true when config is nil or PacingGuard is unset.
func TestPacingGuard_DefaultTrue(t *testing.T) {
	cases := []struct {
		name string
		cfg  *ProjectConfig
	}{
		{"nil config", nil},
		{"empty config", &ProjectConfig{}},
		{"PacingGuard nil", &ProjectConfig{PacingGuard: nil}},
		{"ReminderOnSprintComplete nil", &ProjectConfig{PacingGuard: &PacingGuardConfig{}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.cfg.GetPacingGuardReminderOnSprintComplete(); got != true {
				t.Errorf("expected default true, got %v", got)
			}
		})
	}
}

// TestPacingGuard_ExplicitFalse — explicit false setting returns false.
func TestPacingGuard_ExplicitFalse(t *testing.T) {
	f := false
	cfg := &ProjectConfig{
		PacingGuard: &PacingGuardConfig{ReminderOnSprintComplete: &f},
	}
	if cfg.GetPacingGuardReminderOnSprintComplete() {
		t.Error("expected false when ReminderOnSprintComplete=false")
	}
}

// TestPacingGuard_ExplicitTrue — explicit true setting returns true.
func TestPacingGuard_ExplicitTrue(t *testing.T) {
	tr := true
	cfg := &ProjectConfig{
		PacingGuard: &PacingGuardConfig{ReminderOnSprintComplete: &tr},
	}
	if !cfg.GetPacingGuardReminderOnSprintComplete() {
		t.Error("expected true when ReminderOnSprintComplete=true")
	}
}
