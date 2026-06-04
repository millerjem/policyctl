package audit

import (
	"path/filepath"
	"testing"
)

func TestExpandCheckKeepsUnknownVariables(t *testing.T) {
	check := Check{
		Resource: "arn:${PARTITION}:sqs:${REGION}:${ACCOUNT_ID}:vertex-control-plane/${UNKNOWN}",
	}

	got := ExpandCheck(check, map[string]string{
		"PARTITION":  "aws-us-gov",
		"REGION":     "us-gov-west-1",
		"ACCOUNT_ID": "123456789012",
	})

	want := "arn:aws-us-gov:sqs:us-gov-west-1:123456789012:vertex-control-plane/${UNKNOWN}"
	if got.Resource != want {
		t.Fatalf("Resource = %q, want %q", got.Resource, want)
	}
}

func TestExampleChecksLoad(t *testing.T) {
	files := []string{
		"../../examples/capa-controllers-checks.govcloud.json",
		"../../examples/capa-control-plane-checks.govcloud.json",
		"../../examples/capa-nodes-checks.govcloud.json",
		"../../examples/vertex-permission-checks.govcloud.json",
	}

	vars := map[string]string{
		"PARTITION":  "aws-us-gov",
		"REGION":     "us-gov-west-1",
		"ACCOUNT_ID": "123456789012",
	}

	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			checks, err := LoadChecks(file, vars)
			if err != nil {
				t.Fatalf("LoadChecks(%q): %v", file, err)
			}
			if len(checks) == 0 {
				t.Fatalf("LoadChecks(%q) returned no checks", file)
			}
		})
	}
}
