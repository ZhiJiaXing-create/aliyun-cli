package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunE2EBatchDryRunProbe(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := runE2EBatchDryRun([]string{"--probe"}, strings.NewReader(""), &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "ok" {
		t.Fatalf("stdout = %q, want ok", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunE2EBatchDryRunRejectsNonDryRunCase(t *testing.T) {
	var stdout, stderr bytes.Buffer
	input := strings.NewReader(`{"id":"unsafe","argv":["ecs","DeleteInstance","--InstanceId","i-123"]}` + "\n")

	code := runE2EBatchDryRun(nil, input, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("batch command exit code = %d, want 0; stderr=%q", code, stderr.String())
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if !strings.Contains(stdout.String(), `"id":"unsafe"`) {
		t.Fatalf("stdout = %q, want response id", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"returncode":2`) {
		t.Fatalf("stdout = %q, want safety validation returncode", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"error":"missing_dry_run"`) {
		t.Fatalf("stdout = %q, want missing_dry_run", stdout.String())
	}
}

func TestRunE2EBatchDryRunKeepsWorkerAliveAfterCommandError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var stdout, stderr bytes.Buffer
	input := strings.NewReader(
		`{"id":"bad","argv":["not-a-product","DescribeRegions","--dryrun","--endpoint","example.com"]}` + "\n" +
			`{"id":"still-runs","argv":["not-a-product","DescribeRegions","--dryrun","--endpoint","example.com"]}` + "\n",
	)

	code := runE2EBatchDryRun(nil, input, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("batch command exit code = %d, want 0; stderr=%q", code, stderr.String())
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if strings.Count(stdout.String(), "\n") != 2 {
		t.Fatalf("stdout = %q, want one JSON line per request", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"id":"bad"`) || !strings.Contains(stdout.String(), `"id":"still-runs"`) {
		t.Fatalf("stdout = %q, want both request ids", stdout.String())
	}
	if strings.Contains(stdout.String(), `"returncode":0`) {
		t.Fatalf("stdout = %q, want captured non-zero command error returncode", stdout.String())
	}
}
