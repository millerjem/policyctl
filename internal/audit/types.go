package audit

import "strings"

const (
	DecisionAllowed      = "allowed"
	DecisionExplicitDeny = "explicitDeny"
	DecisionImplicitDeny = "implicitDeny"
)

type Check struct {
	Name     string         `json:"name"`
	Action   string         `json:"action"`
	Resource string         `json:"resource"`
	Expected string         `json:"expected"`
	Context  []ContextEntry `json:"context,omitempty"`
}

type ContextEntry struct {
	Name   string   `json:"name"`
	Type   string   `json:"type"`
	Values []string `json:"values"`
}

type CheckGroup struct {
	Title  string
	Source string
	Checks []Check
}

type Result struct {
	Check                Check
	Decision             string
	MissingContextValues []string
}

type ResultGroup struct {
	Title   string
	Source  string
	Results []Result
}

type Row struct {
	Status         string
	Name           string
	Action         string
	Resource       string
	Expected       string
	Decision       string
	MissingContext string
}

func IsSatisfied(expected, decision string) bool {
	expected = strings.TrimSpace(expected)
	decision = strings.TrimSpace(decision)

	switch expected {
	case DecisionAllowed:
		return decision == DecisionAllowed
	case "denied":
		return decision != DecisionAllowed
	case DecisionExplicitDeny, DecisionImplicitDeny:
		return decision == expected
	default:
		return decision == expected
	}
}

func AllSatisfied(results []Result) bool {
	for _, result := range results {
		if !IsSatisfied(result.Check.Expected, result.Decision) {
			return false
		}
	}
	return true
}

func AllGroupsSatisfied(groups []ResultGroup) bool {
	for _, group := range groups {
		if !AllSatisfied(group.Results) {
			return false
		}
	}
	return true
}

func RowsFromResults(results []Result, failuresOnly bool) []Row {
	rows := make([]Row, 0, len(results))
	for _, result := range results {
		status := "FAIL"
		if IsSatisfied(result.Check.Expected, result.Decision) {
			status = "PASS"
		}
		if failuresOnly && status == "PASS" {
			continue
		}

		missing := "-"
		if len(result.MissingContextValues) > 0 {
			missing = strings.Join(result.MissingContextValues, ", ")
		}

		rows = append(rows, Row{
			Status:         status,
			Name:           result.Check.Name,
			Action:         result.Check.Action,
			Resource:       result.Check.Resource,
			Expected:       result.Check.Expected,
			Decision:       result.Decision,
			MissingContext: missing,
		})
	}
	return rows
}
