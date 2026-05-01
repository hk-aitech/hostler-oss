package sprint

import (
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/domain"
)

func TestParseTier(t *testing.T) {
	cases := []struct {
		in       string
		want     SprintTier
		explicit bool
	}{
		{"solo", TierSolo, true},
		{"standard", TierStandard, true},
		{"pair", TierPair, true},
		{"squad", TierSquad, true},
		{"  STANDARD  ", TierStandard, true},
		{"", TierSolo, false},
		{"unknown", TierSolo, false},
	}
	for _, c := range cases {
		got, ok := ParseTier(c.in)
		if got != c.want || ok != c.explicit {
			t.Errorf("ParseTier(%q) = (%q, %v), want (%q, %v)", c.in, got, ok, c.want, c.explicit)
		}
	}
}

func TestEstimateToPoints(t *testing.T) {
	cases := []struct {
		est  domain.Estimate
		want int
	}{
		{domain.EstimateXS, 1},
		{domain.EstimateS, 2},
		{domain.EstimateM, 3},
		{domain.EstimateL, 5},
		{domain.EstimateXL, 8},
		{domain.Estimate(""), 0},
		{domain.Estimate("XXL"), 0},
	}
	for _, c := range cases {
		if got := EstimateToPoints(c.est); got != c.want {
			t.Errorf("EstimateToPoints(%q) = %d, want %d", c.est, got, c.want)
		}
	}
}

func TestSumPoints(t *testing.T) {
	got := SumPoints([]domain.Estimate{domain.EstimateXS, domain.EstimateS, domain.EstimateM, domain.EstimateL, domain.EstimateXL})
	want := 1 + 2 + 3 + 5 + 8
	if got != want {
		t.Errorf("SumPoints = %d, want %d", got, want)
	}
}

func TestDefaultsFor(t *testing.T) {
	cases := []struct {
		tier      SprintTier
		minTask   int
		maxTask   int
		maxCap    int
	}{
		{TierSolo, 1, 3, 8},
		{TierStandard, 3, 7, 20},
		{TierPair, 7, 15, 50},
		{TierSquad, 15, 0, 0},
	}
	for _, c := range cases {
		d := DefaultsFor(c.tier)
		if d.MinTaskCount != c.minTask || d.MaxTaskCount != c.maxTask || d.MaxCapacity != c.maxCap {
			t.Errorf("DefaultsFor(%q) = %+v, want min=%d max=%d cap=%d",
				c.tier, d, c.minTask, c.maxTask, c.maxCap)
		}
	}
}

func TestCheckTier_Solo(t *testing.T) {
	cases := []struct {
		name    string
		ests    []domain.Estimate
		blocked bool
	}{
		// solo: 1~3 tasks, ≤8 pt
		{"solo normal (1 task XS)", []domain.Estimate{domain.EstimateXS}, false},
		{"solo normal boundary (3 task M+M+S=8pt)", []domain.Estimate{domain.EstimateM, domain.EstimateM, domain.EstimateS}, false},
		{"solo violation (4 tasks)", []domain.Estimate{domain.EstimateXS, domain.EstimateXS, domain.EstimateXS, domain.EstimateXS}, true},
		{"solo violation (capacity 9pt)", []domain.Estimate{domain.EstimateL, domain.EstimateM, domain.EstimateXS}, true},
		{"solo violation (0 tasks)", []domain.Estimate{}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := CheckTier(TierSolo, c.ests, TierDefaults{})
			if r.IsBlocked() != c.blocked {
				t.Errorf("IsBlocked = %v (violations=%v), want %v", r.IsBlocked(), r.Violations, c.blocked)
			}
		})
	}
}

func TestCheckTier_Standard(t *testing.T) {
	cases := []struct {
		name    string
		ests    []domain.Estimate
		blocked bool
	}{
		// standard: 3~7 tasks, ≤20 pt
		{"standard normal boundary (3 tasks)", []domain.Estimate{domain.EstimateS, domain.EstimateS, domain.EstimateS}, false},
		{"standard normal boundary (7 tasks 14pt)", []domain.Estimate{domain.EstimateS, domain.EstimateS, domain.EstimateS, domain.EstimateS, domain.EstimateS, domain.EstimateS, domain.EstimateS}, false},
		{"standard violation (2 tasks < min 3)", []domain.Estimate{domain.EstimateS, domain.EstimateS}, true},
		{"standard violation (8 tasks > max 7)", []domain.Estimate{domain.EstimateS, domain.EstimateS, domain.EstimateS, domain.EstimateS, domain.EstimateS, domain.EstimateS, domain.EstimateS, domain.EstimateS}, true},
		{"standard violation (capacity 21pt)", []domain.Estimate{domain.EstimateXL, domain.EstimateXL, domain.EstimateL}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := CheckTier(TierStandard, c.ests, TierDefaults{})
			if r.IsBlocked() != c.blocked {
				t.Errorf("IsBlocked = %v (violations=%v), want %v", r.IsBlocked(), r.Violations, c.blocked)
			}
		})
	}
}

func TestCheckTier_Squad_Unbounded(t *testing.T) {
	// squad: min=15, max=0 (unlimited), maxCap=0 (unlimited)
	ests := make([]domain.Estimate, 50)
	for i := range ests {
		ests[i] = domain.EstimateXL
	}
	r := CheckTier(TierSquad, ests, TierDefaults{})
	if r.IsBlocked() {
		t.Errorf("squad 50 tasks * XL=400pt should pass (max unlimited), got violations=%v", r.Violations)
	}
	if r.MaxTaskCount != 0 || r.MaxCapacity != 0 {
		t.Errorf("squad max should be 0 (unlimited), got max=%d cap=%d", r.MaxTaskCount, r.MaxCapacity)
	}
}

func TestCheckTier_Squad_MinViolation(t *testing.T) {
	// squad min=15, 14 tasks → BLOCK
	ests := make([]domain.Estimate, 14)
	for i := range ests {
		ests[i] = domain.EstimateXS
	}
	r := CheckTier(TierSquad, ests, TierDefaults{})
	if !r.IsBlocked() {
		t.Errorf("squad 14 tasks < min 15 should BLOCK, got pass")
	}
}

func TestCheckTier_Override(t *testing.T) {
	// solo default max=3, extended via override to max=5
	ests := []domain.Estimate{
		domain.EstimateXS, domain.EstimateXS, domain.EstimateXS,
		domain.EstimateXS, domain.EstimateXS,
	}
	override := TierDefaults{MaxTaskCount: 5, MaxCapacity: 10}
	r := CheckTier(TierSolo, ests, override)
	if r.IsBlocked() {
		t.Errorf("solo override max=5 should pass 5 tasks, got violations=%v", r.Violations)
	}
	if r.MaxTaskCount != 5 || r.MaxCapacity != 10 {
		t.Errorf("override should apply, got max=%d cap=%d", r.MaxTaskCount, r.MaxCapacity)
	}
}
