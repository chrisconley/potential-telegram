package main

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

func TestLLMTokensOutput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("go", "run", ".")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("go run .: %v\nstderr: %s", err, stderr.String())
	}

	got := strings.TrimRight(stdout.String(), "\n")
	want := strings.Join([]string{
		"event completion_42 -> 1 MeterRecord(s)",
		"  record completion_42 contains 2 observations:",
		"    450 input-tokens",
		"    890 output-tokens",
	}, "\n")
	if got != want {
		t.Errorf("examples/README output drift\n got:  %q\n want: %q", got, want)
	}
}
