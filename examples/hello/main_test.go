package main

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

// TestHelloOutput runs the hello example exactly as the README instructs
// and asserts the single output line. If anyone changes the API or the
// math in a way that shifts the displayed number, this test breaks before
// the README's claimed output can drift.
func TestHelloOutput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("go", "run", ".")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("go run .: %v\nstderr: %s", err, stderr.String())
	}

	got := strings.TrimRight(stdout.String(), "\n")
	want := "customer:acme-corp used 11.67 seats (time-weighted-avg) from 2024-01-01 to 2024-01-31"
	if got != want {
		t.Errorf("README quick-start output drift\n got:  %q\n want: %q", got, want)
	}
}
