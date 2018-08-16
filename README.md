# Elastic CI Stack for AWS: ECS Edition (2 elastic 2 stack)

This is an **experimental version** of our main [AWS stack](https://github.com/buildkite/elastic-ci-stack-for-aws) that makes use of ECS and Spot Fleets.

## Design

* An AWS SpotFleet is used to run ECS Instances in a dedicated ECS cluster.
* An ECS Service is used to run our Agent via docker.
* A lambda ([buildkite-agent-scaler](https://github.com/buildkite/buildkite-agent-scaler)) is run on a schedule to adjust the capacity of the SpotFleet based on scheduled jobs.
* ECS Instances bootstrap via user-data from a vanilla Amazon ECS AMI

## Open Questions

* Does bootstrapping vanilla ECS mean spin-up is slower?
* How does ECS scheduling handle builds that create docker containers outside of ECS?

## Running

```bash
## Create an Elastic Stack
aws cloudformation create-stack \
  --output text \
  --stack-name buildkite-elastic-stack-ecs \
  --template-body "file://$PWD/templates/elastic-stack.yaml" \
  --capabilities CAPABILITY_IAM \
  --parameters "ParameterKey=BuildkiteAgentToken,ParameterValue=xxx"

## Figure out what availability zones are available
aws ec2 describe-availability-zones \
  --query 'AvailabilityZones[?State==`available`].ZoneName'
[
    "us-east-1a",
    "us-east-1b",
    "us-east-1c",
    "us-east-1d",
    "us-east-1e",
    "us-east-1f"
]

## Create a VPC stack
aws cloudformation create-stack \
  --output text \
  --stack-name buildkite-elastic-stack-ecs-vpc \
  --template-body "file://$PWD/templates/vpc.yaml" \
  --parameters "ParameterKey=AvailabilityZones,ParameterValue=us-east-1a\\,us-east-1b\\,us-east-1c\\,us-east-1d\\,us-east-1e\\,us-east-1f" \
               "ParameterKey=SubnetConfiguration,ParameterValue=6 private subnets + 2 public subnets with NAT Gateways for internet access"

## Get Private Subnets and Vpc from VPC stack
aws cloudformation describe-stacks \
  --stack-name buildkite-elastic-stack-ecs-vpc \
  --query "Stacks[*].Outputs"



```
