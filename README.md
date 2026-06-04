# policyctl

`policyctl` runs IAM policy simulations for a list of expected permission checks and prints a wrapped box table of PASS/FAIL rows. PASS is printed in green and FAIL is printed in red. It can also print a simple true/false go/no-go result.

It uses the AWS CLI rather than the AWS SDK, so it works with the same `AWS_PROFILE`, `AWS_REGION`, SSO session, or GovCloud credentials you already use.

While it is running in an interactive terminal, `policyctl` prints a stderr spinner like `Checking policies 7/25 |`. The final table or boolean result still prints to stdout.

## Build

```bash
go build ./cmd/policyctl
```

## Generate CAPA IAM Policies

CAPA normally creates the IAM roles and managed policies with `clusterawsadm`:

```bash
export AWS_PROFILE=<profile>
export AWS_REGION=us-gov-west-1

clusterawsadm bootstrap iam create-cloudformation-stack
```

You can also print the expected CAPA managed policy templates locally. These document names come from `clusterawsadm bootstrap iam print-policy`:

```bash
mkdir -p generated

clusterawsadm bootstrap iam print-policy \
  --document AWSIAMManagedPolicyControllers \
  > generated/expected-controllers.cfn.json

clusterawsadm bootstrap iam print-policy \
  --document AWSIAMManagedPolicyCloudProviderControlPlane \
  > generated/expected-control-plane.cfn.json

clusterawsadm bootstrap iam print-policy \
  --document AWSIAMManagedPolicyCloudProviderNodes \
  > generated/expected-nodes.cfn.json
```

Those files are CloudFormation managed-policy resources. To fetch the live managed policy documents from GovCloud IAM:

```bash
ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
PARTITION=$(aws sts get-caller-identity --query Arn --output text | cut -d: -f2)

for policy in \
  controllers.cluster-api-provider-aws.sigs.k8s.io \
  control-plane.cluster-api-provider-aws.sigs.k8s.io \
  nodes.cluster-api-provider-aws.sigs.k8s.io
do
  arn="arn:${PARTITION}:iam::${ACCOUNT_ID}:policy/${policy}"
  version=$(aws iam get-policy \
    --policy-arn "${arn}" \
    --query 'Policy.DefaultVersionId' \
    --output text)

  aws iam get-policy-version \
    --policy-arn "${arn}" \
    --version-id "${version}" \
    --query 'PolicyVersion.Document' \
    > "generated/actual-${policy}.json"
done
```

CAPA docs:

- [`clusterawsadm bootstrap iam create-cloudformation-stack`](https://cluster-api-aws.sigs.k8s.io/clusterawsadm/clusterawsadm_bootstrap_iam_create-cloudformation-stack)
- [`clusterawsadm bootstrap iam print-policy`](https://cluster-api-aws.sigs.k8s.io/clusterawsadm/clusterawsadm_bootstrap_iam_print-policy)
- [CAPA IAM permissions](https://cluster-api-aws.sigs.k8s.io/topics/iam-permissions)

## GovCloud CAPA Role Examples

The CAPA example check files are positive sanity checks derived from the generated CAPA managed policies. They are intended to catch missing expected permissions. Add `expected: "denied"` checks separately when you want to enforce explicit hardening expectations.

```bash
export AWS_PROFILE=<profile>
export AWS_REGION=us-gov-west-1

ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
PARTITION=$(aws sts get-caller-identity --query Arn --output text | cut -d: -f2)
```

Run checks against the CAPA controller role:

```bash
go run ./cmd/policyctl \
  --checks examples/capa-controllers-checks.govcloud.json \
  --policy-source-arn "arn:${PARTITION}:iam::${ACCOUNT_ID}:role/controllers.cluster-api-provider-aws.sigs.k8s.io" \
  --region "${AWS_REGION}"
```

Run checks against the CAPA control-plane role:

```bash
go run ./cmd/policyctl \
  --checks examples/capa-control-plane-checks.govcloud.json \
  --policy-source-arn "arn:${PARTITION}:iam::${ACCOUNT_ID}:role/control-plane.cluster-api-provider-aws.sigs.k8s.io" \
  --region "${AWS_REGION}"
```

Run checks against the CAPA nodes role:

```bash
go run ./cmd/policyctl \
  --checks examples/capa-nodes-checks.govcloud.json \
  --policy-source-arn "arn:${PARTITION}:iam::${ACCOUNT_ID}:role/nodes.cluster-api-provider-aws.sigs.k8s.io" \
  --region "${AWS_REGION}"
```

Print only deficiencies for any role by adding `--failures-only`:

```bash
go run ./cmd/policyctl \
  --checks examples/capa-controllers-checks.govcloud.json \
  --policy-source-arn "arn:${PARTITION}:iam::${ACCOUNT_ID}:role/controllers.cluster-api-provider-aws.sigs.k8s.io" \
  --region "${AWS_REGION}" \
  --failures-only
```

Run all three examples:

```bash
for role in controllers control-plane nodes
do
  go run ./cmd/policyctl \
    --checks "examples/capa-${role}-checks.govcloud.json" \
    --policy-source-arn "arn:${PARTITION}:iam::${ACCOUNT_ID}:role/${role}.cluster-api-provider-aws.sigs.k8s.io" \
    --region "${AWS_REGION}" \
    --failures-only
done
```

## Go/No-Go Outcome

Use boolean output when you only want to know whether the role/policy satisfies every check:

```bash
go run ./cmd/policyctl \
  --checks examples/capa-controllers-checks.govcloud.json \
  --policy-source-arn "arn:${PARTITION}:iam::${ACCOUNT_ID}:role/controllers.cluster-api-provider-aws.sigs.k8s.io" \
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
  --checks examples/capa-controllers-checks.govcloud.json \
  --policy-source-arn "arn:${PARTITION}:iam::${ACCOUNT_ID}:role/controllers.cluster-api-provider-aws.sigs.k8s.io" \
  --region "${AWS_REGION}" \
  --output bool \
  --fail-on-deficiency
```

To disable the progress spinner in scripts, add `--no-progress`.

You can use the same boolean mode for a raw policy document:

```bash
go run ./cmd/policyctl \
  --checks examples/capa-controllers-checks.govcloud.json \
  --policy-document generated/actual-controllers.cluster-api-provider-aws.sigs.k8s.io.json \
  --partition aws-us-gov \
  --region us-gov-west-1 \
  --account-id "${ACCOUNT_ID}" \
  --output bool
```

## Custom policy document simulation

This is useful when you want to evaluate a policy JSON file without depending on a role attachment:

```bash
go run ./cmd/policyctl \
  --checks examples/capa-controllers-checks.govcloud.json \
  --policy-document generated/actual-controllers.cluster-api-provider-aws.sigs.k8s.io.json \
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
