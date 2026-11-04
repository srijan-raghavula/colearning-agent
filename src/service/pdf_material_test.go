package service

import (
	"strings"
	"testing"
)

func TestExtractPDFTextRejectsOversizedInput(t *testing.T) {
	_, err := ExtractPDFText(strings.NewReader("x"), MaxPDFMaterialBytes+1)
	if err == nil {
		t.Fatal("expected oversized PDF to be rejected")
	}
}

func TestSafePDFTitle(t *testing.T) {
	if got := SafePDFTitle("../lesson-notes.pdf"); got != "lesson-notes" {
		t.Fatalf("unexpected title %q", got)
	}
	if got := SafePDFTitle(".pdf"); got != "Uploaded PDF material" {
		t.Fatalf("unexpected fallback title %q", got)
	}
}
