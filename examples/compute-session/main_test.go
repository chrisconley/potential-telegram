package main

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

func TestComputeSessionOutput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("go", "run", ".")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("go run .: %v\nstderr: %s", err, stderr.String())
	}

	got := strings.TrimRight(stdout.String(), "\n")
	want := "customer:acme used 17 compute-hours across 3 sessions on 2024-01-15"
	if got != want {
		t.Errorf("examples/README output drift\n got:  %q\n want: %q", got, want)
	}
}
