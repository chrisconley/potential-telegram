package main

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

func TestConditionalTierOutput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("go", "run", ".")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("go run .: %v\nstderr: %s", err, stderr.String())
	}

	got := strings.TrimRight(stdout.String(), "\n")
	want := strings.Join([]string{
		"metered 2/3 events into records (free-tier filtered out)",
		"sum premium-tokens for the day: 1300 (from 2 events)",
	}, "\n")
	if got != want {
		t.Errorf("examples/README output drift\n got:  %q\n want: %q", got, want)
	}
}
