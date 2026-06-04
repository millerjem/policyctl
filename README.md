# policyctl

`policyctl` runs IAM policy simulations for a list of expected permission checks and prints a wrapped box table of PASS/FAIL rows. PASS is printed in green and FAIL is printed in red. It can also print a simple true/false go/no-go result.

It uses the AWS CLI rather than the AWS SDK, so it works with the same `AWS_PROFILE`, `AWS_REGION`, SSO session, or GovCloud credentials you already use.

While it is running in an interactive terminal, `policyctl` prints a stderr spinner like `Checking policies 7/25 |`. The final table or boolean result still prints to stdout.

## Build

```bash
go build ./cmd/policyctl
```

## GovCloud CAPA Controller Role

```bash
export AWS_PROFILE=<profile>
export AWS_REGION=us-gov-west-1

ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)

go run ./cmd/policyctl \
  --checks examples/vertex-permission-checks.govcloud.json \
  --policy-source-arn "arn:aws-us-gov:iam::${ACCOUNT_ID}:role/controllers.cluster-api-provider-aws.sigs.k8s.io" \
  --region "${AWS_REGION}"
```

Print only deficiencies against that same GovCloud CAPA controller role:

```bash
go run ./cmd/policyctl \
  --checks examples/vertex-permission-checks.govcloud.json \
  --policy-source-arn "arn:aws-us-gov:iam::${ACCOUNT_ID}:role/controllers.cluster-api-provider-aws.sigs.k8s.io" \
  --region "${AWS_REGION}" \
  --failures-only
```

## Go/No-Go Outcome

Use boolean output when you only want to know whether the role/policy satisfies every check:

```bash
go run ./cmd/policyctl \
  --checks examples/vertex-permission-checks.govcloud.json \
  --policy-source-arn "arn:aws-us-gov:iam::${ACCOUNT_ID}:role/controllers.cluster-api-provider-aws.sigs.k8s.io" \
  --region "${AWS_REGION}" \
  --output bool
```

This prints:

```text
true
```

when every check passes, or:

```text
false
```

when one or more deficiencies exist.

For CI or script usage, add `--fail-on-deficiency` so a no-go result exits with status `1`:

```bash
go run ./cmd/policyctl \
  --checks examples/vertex-permission-checks.govcloud.json \
  --policy-source-arn "arn:aws-us-gov:iam::${ACCOUNT_ID}:role/controllers.cluster-api-provider-aws.sigs.k8s.io" \
  --region "${AWS_REGION}" \
  --output bool \
  --fail-on-deficiency
```

To disable the progress spinner in scripts, add `--no-progress`.

You can use the same boolean mode for a raw policy document:

```bash
go run ./cmd/policyctl \
  --checks examples/vertex-permission-checks.govcloud.json \
  --policy-document actual-controllers.json \
  --partition aws-us-gov \
  --region us-gov-west-1 \
  --account-id "${ACCOUNT_ID}" \
  --output bool
```

## Custom policy document simulation

This is useful when you want to evaluate a policy JSON file without depending on a role attachment:

```bash
go run ./cmd/policyctl \
  --checks examples/vertex-permission-checks.govcloud.json \
  --policy-document actual-controllers.json \
  --partition aws-us-gov \
  --region us-gov-west-1 \
  --account-id 123456789012
```

## Check format

```json
[
  {
    "name": "EC2 describe subnets",
    "action": "ec2:DescribeSubnets",
    "resource": "*",
    "expected": "allowed"
  },
  {
    "name": "Restricted CloudTrail bucket denied",
    "action": "s3:GetObject",
    "resource": "arn:${PARTITION}:s3:::cloudtrail-audit/logfile",
    "expected": "denied"
  }
]
```

Supported substitutions are `${PARTITION}`, `${REGION}`, and `${ACCOUNT_ID}`. You can pass them as flags, set environment variables, or let the tool discover account and partition from `aws sts get-caller-identity`.
