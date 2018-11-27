#!/bin/bash
set -euo pipefail

LAMBDA_BUCKET=buildkite-aws-stack-ecs-dev
DOCKER_TAG=lox24/buildkite-agent-ecs
VPC_STACK=${VPC_STACK:-}
STACK_SUFFIX=dev
QUEUE=dev

if [[ -z "${VPC_STACK:-}" ]] ; then
  export VPC_STACK=buildkite-vpc-${STACK_SUFFIX}

  ## Figure out what availability zones are available
  export EC2_AVAILABILITY_ZONES=$(aws ec2 describe-availability-zones \
    --query 'AvailabilityZones[?State==`available`].ZoneName' \
    --output text | sed -E -e 's/[[:blank:]]+/,/g')

  export EC2_AVAILABILITY_ZONES_COUNT=$(awk -F, '{print NF-1}' <<< "$EC2_AVAILABILITY_ZONES")

  ## Create a VPC stack
  echo "~~~ Creating ${VPC_STACK}"
  parfait create-stack \
    -t templates/vpc/template.yaml \
    "$VPC_STACK" \
    "AvailabilityZones=${EC2_AVAILABILITY_ZONES}" \
    "SubnetConfiguration=${EC2_AVAILABILITY_ZONES_COUNT} public subnets"
fi

## Get Private Subnets and Vpc from VPC stack
export EC2_VPC_ID="$(aws cloudformation describe-stacks \
  --stack-name "$VPC_STACK" \
  --query 'Stacks[0].Outputs[?OutputKey==`Vpc`].OutputValue' \
  --output text)"

export EC2_VPC_SUBNETS="$(aws cloudformation describe-stacks \
  --stack-name "$VPC_STACK" \
  --query 'Stacks[0].Outputs[?OutputKey==`PublicSubnets`].OutputValue' \
  --output text)"

if ! aws cloudformation describe-stacks --stack-name buildkite-agent-${STACK_SUFFIX} &> /dev/null ; then
  echo "~~~ Creating buildkite-agent-${STACK_SUFFIX}"
  parfait create-stack \
    -t templates/agent/template.yaml \
    buildkite-agent-${STACK_SUFFIX} \
    "LambdaBucket=${LAMBDA_BUCKET}" \
    "AgentDockerImage=${DOCKER_TAG}" \
    "AgentBootstrapDockerImage=${DOCKER_TAG}" \
    "BuildkiteQueue=${QUEUE}"
fi

if ! aws cloudformation describe-stacks --stack-name buildkite-spotfleet-${STACK_SUFFIX} &> /dev/null ; then
  echo "~~~ Creating buildkite-spotfleet-${STACK_SUFFIX}"
  parfait create-stack \
    -t templates/compute/spotfleet/template.yaml \
    buildkite-spotfleet-${STACK_SUFFIX} \
    "VPC=${EC2_VPC_ID?}" \
    "Subnets=${EC2_VPC_SUBNETS}" \
    "ECSCluster=buildkite-agent-${STACK_SUFFIX}"
fi
