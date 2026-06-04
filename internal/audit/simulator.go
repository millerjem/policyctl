package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type Simulator interface {
	Simulate(context.Context, Check) (Result, error)
}

type AWSCLIOptions struct {
	Binary  string
	Profile string
	Region  string
}

type AWSSimulator struct {
	Options         AWSCLIOptions
	PolicySourceARN string
	PolicyDocument  string
}

type Caller struct {
	AccountID string
	Partition string
}

type ProgressFunc func(done, total int)

func RunChecks(ctx context.Context, checks []Check, simulator Simulator) ([]Result, error) {
	return RunChecksWithProgress(ctx, checks, simulator, nil)
}

func RunChecksWithProgress(ctx context.Context, checks []Check, simulator Simulator, progress ProgressFunc) ([]Result, error) {
	results := make([]Result, 0, len(checks))
	total := len(checks)
	if progress != nil {
		progress(0, total)
	}

	for _, check := range checks {
		result, err := simulator.Simulate(ctx, check)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
		if progress != nil {
			progress(len(results), total)
		}
	}
	return results, nil
}

func (s AWSSimulator) Simulate(ctx context.Context, check Check) (Result, error) {
	args := []string{"iam"}
	if s.PolicySourceARN != "" {
		args = append(args,
			"simulate-principal-policy",
			"--policy-source-arn", s.PolicySourceARN,
		)
	} else {
		args = append(args,
			"simulate-custom-policy",
			"--policy-input-list", "file://"+s.PolicyDocument,
		)
	}

	args = append(args,
		"--action-names", check.Action,
		"--resource-arns", check.Resource,
		"--output", "json",
		"--no-cli-pager",
	)

	if len(check.Context) > 0 {
		args = append(args, "--context-entries")
		for _, entry := range check.Context {
			args = append(args, contextEntryArg(entry))
		}
	}
	args = appendAWSOptions(args, s.Options)

	output, err := runAWS(ctx, s.Options.Binary, args)
	if err != nil {
		return Result{}, fmt.Errorf("simulate %s on %s: %w", check.Action, check.Resource, err)
	}

	var response simulateResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return Result{}, fmt.Errorf("parse simulator response for %s: %w", check.Name, err)
	}
	if len(response.EvaluationResults) == 0 {
		return Result{}, fmt.Errorf("simulator returned no evaluation results for %s", check.Name)
	}

	eval := response.EvaluationResults[0]
	return Result{
		Check:                check,
		Decision:             eval.EvalDecision,
		MissingContextValues: eval.MissingContextValues,
	}, nil
}

func DiscoverCaller(ctx context.Context, opts AWSCLIOptions) (Caller, error) {
	args := appendAWSOptions([]string{
		"sts",
		"get-caller-identity",
		"--output", "json",
		"--no-cli-pager",
	}, opts)

	output, err := runAWS(ctx, opts.Binary, args)
	if err != nil {
		return Caller{}, err
	}

	var response struct {
		Account string `json:"Account"`
		Arn     string `json:"Arn"`
	}
	if err := json.Unmarshal(output, &response); err != nil {
		return Caller{}, err
	}

	parts := strings.Split(response.Arn, ":")
	if len(parts) < 2 {
		return Caller{}, fmt.Errorf("caller ARN did not include a partition: %s", response.Arn)
	}

	return Caller{
		AccountID: response.Account,
		Partition: parts[1],
	}, nil
}

func contextEntryArg(entry ContextEntry) string {
	return fmt.Sprintf(
		"ContextKeyName=%s,ContextKeyValues=%s,ContextKeyType=%s",
		entry.Name,
		strings.Join(entry.Values, ","),
		entry.Type,
	)
}

func appendAWSOptions(args []string, opts AWSCLIOptions) []string {
	if opts.Profile != "" {
		args = append(args, "--profile", opts.Profile)
	}
	if opts.Region != "" {
		args = append(args, "--region", opts.Region)
	}
	return args
}

func runAWS(ctx context.Context, binary string, args []string) ([]byte, error) {
	if binary == "" {
		binary = "aws"
	}

	cmd := exec.CommandContext(ctx, binary, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s %s failed: %w\n%s", binary, strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

type simulateResponse struct {
	EvaluationResults []evaluationResult `json:"EvaluationResults"`
}

type evaluationResult struct {
	EvalActionName       string   `json:"EvalActionName"`
	EvalResourceName     string   `json:"EvalResourceName"`
	EvalDecision         string   `json:"EvalDecision"`
	MissingContextValues []string `json:"MissingContextValues"`
}
