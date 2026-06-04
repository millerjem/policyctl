package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func LoadChecks(path string, vars map[string]string) ([]Check, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read checks: %w", err)
	}

	var checks []Check
	if err := json.Unmarshal(data, &checks); err != nil {
		return nil, fmt.Errorf("parse checks: %w", err)
	}

	for i := range checks {
		checks[i] = ExpandCheck(checks[i], vars)
		if err := validateCheck(checks[i], i); err != nil {
			return nil, err
		}
	}

	return checks, nil
}

func ExpandCheck(check Check, vars map[string]string) Check {
	check.Name = expandVars(check.Name, vars)
	check.Action = expandVars(check.Action, vars)
	check.Resource = expandVars(check.Resource, vars)
	check.Expected = expandVars(check.Expected, vars)
	for i := range check.Context {
		check.Context[i].Name = expandVars(check.Context[i].Name, vars)
		check.Context[i].Type = expandVars(check.Context[i].Type, vars)
		for j := range check.Context[i].Values {
			check.Context[i].Values[j] = expandVars(check.Context[i].Values[j], vars)
		}
	}
	return check
}

func VariablesFromEnv() map[string]string {
	vars := map[string]string{
		"ACCOUNT_ID": os.Getenv("ACCOUNT_ID"),
		"PARTITION":  os.Getenv("PARTITION"),
		"REGION":     os.Getenv("AWS_REGION"),
	}
	if vars["ACCOUNT_ID"] == "" {
		vars["ACCOUNT_ID"] = os.Getenv("AWS_ACCOUNT_ID")
	}
	if vars["REGION"] == "" {
		vars["REGION"] = os.Getenv("AWS_DEFAULT_REGION")
	}
	return vars
}

func expandVars(value string, vars map[string]string) string {
	return os.Expand(value, func(name string) string {
		if value, ok := vars[name]; ok && value != "" {
			return value
		}
		return "${" + name + "}"
	})
}

func validateCheck(check Check, index int) error {
	missing := make([]string, 0, 4)
	if strings.TrimSpace(check.Name) == "" {
		missing = append(missing, "name")
	}
	if strings.TrimSpace(check.Action) == "" {
		missing = append(missing, "action")
	}
	if strings.TrimSpace(check.Resource) == "" {
		missing = append(missing, "resource")
	}
	if strings.TrimSpace(check.Expected) == "" {
		missing = append(missing, "expected")
	}
	if len(missing) > 0 {
		return fmt.Errorf("check %d missing required fields: %s", index, strings.Join(missing, ", "))
	}
	return nil
}
