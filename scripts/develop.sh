#!/bin/bash
set -euo pipefail

# Used in development of the stack. Updates the docker image and updates
# stacks locally

LAMBDA_BUCKET=buildkite-aws-stack-ecs-dev
DOCKER_TAG=lox24/buildkite-agent-ecs
STACK_SUFFIX=dev

printf -- '\n--- Updating docker image\n'

make docker DOCKER_TAG=${DOCKER_TAG}
docker push ${DOCKER_TAG}

docker_image="$(docker inspect \
  --format='{{index .RepoDigests 0}}' \
  ${DOCKER_TAG}:latest)"

printf -- '\n--- Updating spotfleet stack\n'

spotfleet_scaler_version="$(aws s3api head-object \
  --bucket ${LAMBDA_BUCKET} \
  --key ecs-spotfleet-scaler.zip --query "VersionId" --output text)"

parfait update-stack \
  -t templates/compute/spotfleet/template.yaml \
  buildkite-spotfleet-${STACK_SUFFIX} \
  "LambdaObjectVersion=${spotfleet_scaler_version}" \
  "LambdaScheduleState=DISABLED"

printf -- '\n--- Updating agent stack'

service_scaler_version="$(aws s3api head-object \
  --bucket ${LAMBDA_BUCKET} \
  --key ecs-service-scaler.zip --query "VersionId" --output text)"

parfait update-stack \
  -t templates/agent/template.yaml \
  buildkite-agent-${STACK_SUFFIX} \
  "AgentDockerImage=${docker_image}" \
  "LambdaObjectVersion=${service_scaler_version}" \
  "MinSize=1" \
  "LambdaScheduleState=DISABLED"

