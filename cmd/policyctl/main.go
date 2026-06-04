package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"policyctl/internal/audit"
)

type exitCode int

func (e exitCode) Error() string {
	return fmt.Sprintf("exit code %d", e)
}

func main() {
	if err := run(); err != nil {
		var code exitCode
		if errors.As(err, &code) {
			os.Exit(int(code))
		}
		fmt.Fprintf(os.Stderr, "policyctl: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		checksPath      string
		policySourceARN string
		policyDocument  string
		awsCLI          string
		profile         string
		region          string
		accountID       string
		partition       string
		output          string
		failuresOnly    bool
		failOnDef       bool
		noDiscover      bool
		noProgress      bool
		timeout         time.Duration
	)

	flag.StringVar(&checksPath, "checks", "", "JSON file containing permission checks")
	flag.StringVar(&policySourceARN, "policy-source-arn", "", "IAM user, group, or role ARN to simulate with aws iam simulate-principal-policy")
	flag.StringVar(&policyDocument, "policy-document", "", "IAM policy document JSON to simulate with aws iam simulate-custom-policy")
	flag.StringVar(&awsCLI, "aws-cli", "aws", "AWS CLI binary")
	flag.StringVar(&profile, "profile", "", "AWS profile to pass to the AWS CLI")
	flag.StringVar(&region, "region", "", "AWS region to pass to the AWS CLI and use for ${REGION}")
	flag.StringVar(&accountID, "account-id", "", "AWS account ID used for ${ACCOUNT_ID}")
	flag.StringVar(&partition, "partition", "", "AWS partition used for ${PARTITION}, for example aws-us-gov")
	flag.StringVar(&output, "output", "table", "output format: table or bool")
	flag.BoolVar(&failuresOnly, "failures-only", false, "print only failing checks")
	flag.BoolVar(&failOnDef, "fail-on-deficiency", false, "exit with status 1 when one or more checks fail")
	flag.BoolVar(&noDiscover, "no-discover-caller", false, "do not call aws sts get-caller-identity to fill ACCOUNT_ID/PARTITION")
	flag.BoolVar(&noProgress, "no-progress", false, "disable spinner progress output")
	flag.DurationVar(&timeout, "timeout", 5*time.Minute, "overall command timeout")
	flag.Parse()

	if checksPath == "" {
		return errors.New("--checks is required")
	}
	if (policySourceARN == "") == (policyDocument == "") {
		return errors.New("provide exactly one of --policy-source-arn or --policy-document")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	vars := audit.VariablesFromEnv()
	if region != "" {
		vars["REGION"] = region
	}
	if accountID != "" {
		vars["ACCOUNT_ID"] = accountID
	}
	if partition != "" {
		vars["PARTITION"] = partition
	}

	if !noDiscover && (vars["ACCOUNT_ID"] == "" || vars["PARTITION"] == "") {
		caller, err := audit.DiscoverCaller(ctx, audit.AWSCLIOptions{
			Binary:  awsCLI,
			Profile: profile,
			Region:  region,
		})
		if err == nil {
			if vars["ACCOUNT_ID"] == "" {
				vars["ACCOUNT_ID"] = caller.AccountID
			}
			if vars["PARTITION"] == "" {
				vars["PARTITION"] = caller.Partition
			}
		}
	}

	checks, err := audit.LoadChecks(checksPath, vars)
	if err != nil {
		return err
	}

	simulator := audit.AWSSimulator{
		Options: audit.AWSCLIOptions{
			Binary:  awsCLI,
			Profile: profile,
			Region:  region,
		},
		PolicySourceARN: policySourceARN,
		PolicyDocument:  policyDocument,
	}

	spinner := audit.NewTerminalSpinner(os.Stderr)
	if noProgress {
		spinner = audit.NewSpinner(os.Stderr, false)
	}
	defer spinner.Stop()

	results, err := audit.RunChecksWithProgress(ctx, checks, simulator, spinner.ProgressFunc())
	if err != nil {
		return err
	}
	spinner.Stop()

	allPassed := audit.AllSatisfied(results)

	switch output {
	case "bool":
		fmt.Println(allPassed)
		if !allPassed && failOnDef {
			return exitCode(1)
		}
		return nil
	case "table":
	default:
		return fmt.Errorf("unsupported --output %q; use table or bool", output)
	}

	rows := audit.RowsFromResults(results, failuresOnly)
	if len(rows) == 0 {
		fmt.Println("No deficiencies found.")
		if !allPassed && failOnDef {
			return exitCode(1)
		}
		return nil
	}

	fmt.Print(audit.RenderTable(rows, audit.DefaultColumns()))
	if !allPassed && failOnDef {
		return exitCode(1)
	}
	return nil
}
