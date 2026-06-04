package audit

import (
	"strings"
	"testing"
)

func TestRenderTableWrapsRows(t *testing.T) {
	rows := []Row{
		{
			Status:         "FAIL",
			Name:           "DynamoDB leader registry update",
			Action:         "dynamodb:UpdateItem",
			Resource:       "arn:aws-us-gov:dynamodb:us-gov-west-1:102015676567:table/vertex-prod-AWSEBWorkerCronLeaderRegistry",
			Expected:       "allowed",
			Decision:       "implicitDeny",
			MissingContext: "-",
		},
	}

	table := RenderTable(rows, DefaultColumns())

	for _, fragment := range []string{
		"╭────────┬",
		"│ STATUS │ NAME",
		"│ \x1b[31mFAIL\x1b[0m   │ DynamoDB leader",
		"implicitDeny",
		"╰────────┴",
	} {
		if !strings.Contains(table, fragment) {
			t.Fatalf("rendered table missing %q:\n%s", fragment, table)
		}
	}
}

func TestDisplayLenIgnoresANSIColor(t *testing.T) {
	value := colorStatus("FAIL")
	if got := displayLen(value); got != 4 {
		t.Fatalf("displayLen(%q) = %d, want 4", value, got)
	}
}

func TestRowsFromResultsCanFilterFailures(t *testing.T) {
	results := []Result{
		{Check: Check{Name: "pass", Expected: "denied"}, Decision: "implicitDeny"},
		{Check: Check{Name: "fail", Expected: "allowed"}, Decision: "implicitDeny"},
	}

	rows := RowsFromResults(results, true)
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	if rows[0].Name != "fail" || rows[0].Status != "FAIL" {
		t.Fatalf("unexpected filtered row: %+v", rows[0])
	}
}
