package audit

import "testing"

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
