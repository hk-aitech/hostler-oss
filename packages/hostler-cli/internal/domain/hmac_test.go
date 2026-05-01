package domain

import "testing"

// T836 ADR-001 A3 — WorkTicket / Signature value-object unit tests.

func TestSignature_IsValid_True(t *testing.T) {
	s := Signature{Algorithm: "sha256", Value: "abcdef"}
	if !s.IsValid() {
		t.Error("a signature with algorithm and value must report IsValid true")
	}
}

func TestSignature_IsValid_MissingAlgorithm(t *testing.T) {
	s := Signature{Value: "abcdef"}
	if s.IsValid() {
		t.Error("a missing algorithm must report IsValid false")
	}
}

func TestSignature_IsValid_MissingValue(t *testing.T) {
	s := Signature{Algorithm: "sha256"}
	if s.IsValid() {
		t.Error("a missing value must report IsValid false")
	}
}

func TestSignature_String_Standard(t *testing.T) {
	s := Signature{Algorithm: "sha256", Value: "abcdef"}
	got := s.String()
	want := "sha256:abcdef"
	if got != want {
		t.Errorf("String = %q, want %q", got, want)
	}
}

func TestSignature_String_MissingAlgorithm(t *testing.T) {
	s := Signature{Value: "abcdef"}
	got := s.String()
	if got != "abcdef" {
		t.Errorf("missing algorithm should return value only: got %q, want abcdef", got)
	}
}

func TestWorkTicket_FieldsPreserved(t *testing.T) {
	wt := WorkTicket{
		TaskID:    TaskID("T834"),
		TicketRef: "WT-T834-abcdef",
		CreatedAt: "2026-04-19",
	}
	if wt.TaskID.String() != "T834" {
		t.Errorf("TaskID not preserved: %s", wt.TaskID)
	}
	if wt.TicketRef != "WT-T834-abcdef" {
		t.Errorf("TicketRef not preserved: %s", wt.TicketRef)
	}
	if wt.CreatedAt != "2026-04-19" {
		t.Errorf("CreatedAt not preserved: %s", wt.CreatedAt)
	}
}
