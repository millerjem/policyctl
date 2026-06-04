package audit

import "testing"

func TestIsSatisfied(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		decision string
		want     bool
	}{
		{name: "allowed passes allowed", expected: "allowed", decision: "allowed", want: true},
		{name: "allowed fails implicit deny", expected: "allowed", decision: "implicitDeny", want: false},
		{name: "denied passes implicit deny", expected: "denied", decision: "implicitDeny", want: true},
		{name: "denied passes explicit deny", expected: "denied", decision: "explicitDeny", want: true},
		{name: "denied fails allowed", expected: "denied", decision: "allowed", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := IsSatisfied(test.expected, test.decision); got != test.want {
				t.Fatalf("IsSatisfied(%q, %q) = %t, want %t", test.expected, test.decision, got, test.want)
			}
		})
	}
}

func TestAllSatisfied(t *testing.T) {
	results := []Result{
		{Check: Check{Expected: "allowed"}, Decision: "allowed"},
		{Check: Check{Expected: "denied"}, Decision: "implicitDeny"},
	}

	if !AllSatisfied(results) {
		t.Fatal("AllSatisfied returned false for all passing results")
	}

	results = append(results, Result{Check: Check{Expected: "allowed"}, Decision: "implicitDeny"})
	if AllSatisfied(results) {
		t.Fatal("AllSatisfied returned true with a failing result")
	}
}
