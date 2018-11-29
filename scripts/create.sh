#!/bin/bash
set -euo pipefail

LAMBDA_BUCKET=buildkite-aws-stack-ecs-dev
DOCKER_IMAGE=lox24/buildkite-agent-ecs
QUEUE=dev
VPC_STACK=${VPC_STACK:-buildkite-vpc}
SPOT_FLEET_STACK=${SPOT_FLEET_STACK:-buildkite-spotfleet}
AGENT_STACK=${AGENT_STACK:-"buildkite-agent-$QUEUE"}

if ! aws cloudformation describe-stacks --stack-name "${VPC_STACK}" &> /dev/null ; then
  ## Figure out what availability zones are available
  EC2_AVAILABILITY_ZONES=$(aws ec2 describe-availability-zones \
    --query 'AvailabilityZones[?State==`available`].ZoneName' \
    --output text | sed -E -e 's/[[:blank:]]+/,/g')

  EC2_AVAILABILITY_ZONES_COUNT=$(awk -F, '{print NF-1}' <<< "$EC2_AVAILABILITY_ZONES")

  ## Create a VPC stack
  echo "~~~ Creating ${VPC_STACK}"
  parfait create-stack \
    -t templates/vpc/template.yaml \
    "$VPC_STACK" \
    "AvailabilityZones=${EC2_AVAILABILITY_ZONES}" \
    "SubnetConfiguration=${EC2_AVAILABILITY_ZONES_COUNT} public subnets"
fi

## Get Private Subnets and Vpc from VPC stack
EC2_VPC_ID="$(aws cloudformation describe-stacks \
  --stack-name "$VPC_STACK" \
  --query 'Stacks[0].Outputs[?OutputKey==`Vpc`].OutputValue' \
  --output text)"

EC2_VPC_SUBNETS="$(aws cloudformation describe-stacks \
  --stack-name "$VPC_STACK" \
  --query 'Stacks[0].Outputs[?OutputKey==`PublicSubnets`].OutputValue' \
  --output text)"

if ! aws cloudformation describe-stacks --stack-name "${SPOT_FLEET_STACK}" &> /dev/null ; then
  echo "~~~ Creating ${SPOT_FLEET_STACK}"
  parfait create-stack \
    -t templates/compute/spotfleet/template.yaml \
    "${SPOT_FLEET_STACK}" \
    "VPC=${EC2_VPC_ID?}" \
    "Subnets=${EC2_VPC_SUBNETS}" \
    "LambdaBucket=${LAMBDA_BUCKET}"
fi

if ! aws cloudformation describe-stacks --stack-name "${AGENT_STACK}" &> /dev/null ; then
  echo "~~~ Creating ${AGENT_STACK}"
  parfait create-stack \
    -t templates/agent/template.yaml \
    "${AGENT_STACK}" \
    "ECSCluster=${SPOT_FLEET_STACK}"
    "AgentDockerImage=${DOCKER_IMAGE}" \
    "BuildkiteQueue=${QUEUE}" \
    "LambdaBucket=${LAMBDA_BUCKET}"
fi

