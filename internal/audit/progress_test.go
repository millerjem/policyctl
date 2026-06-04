package audit

import (
	"bytes"
	"strings"
	"testing"
)

func TestSpinnerPreview(t *testing.T) {
	got := SpinnerPreview(7, 25, "|")
	want := "Checking policies 7/25 |"
	if got != want {
		t.Fatalf("SpinnerPreview() = %q, want %q", got, want)
	}
}

func TestSpinnerWritesProgress(t *testing.T) {
	var out bytes.Buffer
	spinner := NewSpinner(&out, true)
	spinner.Start(25)
	spinner.Advance(7)
	spinner.render(2)
	spinner.Stop()

	got := out.String()
	if !strings.Contains(got, "Checking policies 7/25 |") {
		t.Fatalf("spinner output missing progress, got %q", got)
	}
	if !strings.Contains(got, "\033[2K") {
		t.Fatalf("spinner output did not clear line, got %q", got)
	}
}
