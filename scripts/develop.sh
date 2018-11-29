#!/bin/bash
set -euo pipefail

# Used in development of the stack. Updates the docker image and updates
# stacks locally

DOCKER_AGENT_IMAGE=lox24/buildkite-agent-ecs
DOCKER_SOCKGUARD_IMAGE=lox24/sockguard-ecs

SPOT_FLEET_STACK=${SPOT_FLEET_STACK:-buildkite-spotfleet-dev}
AGENT_STACK=${AGENT_STACK:-"buildkite-agent-dev"}

printf -- '\n--- Updating docker images\n'

make docker docker-push \
  "DOCKER_AGENT_TAG=${DOCKER_AGENT_IMAGE}" \
  "DOCKER_SOCKGUARD_TAG=${DOCKER_SOCKGUARD_IMAGE}"

docker_image="$(docker inspect \
  --format='{{index .RepoDigests 0}}' \
  ${DOCKER_AGENT_IMAGE}:latest)"

printf -- '\n--- Updating spotfleet stack\n'

parfait update-stack \
  -t templates/compute/spotfleet/template.yaml \
  "$SPOT_FLEET_STACK" \
  "MinSize=1" \
  "LambdaScheduleState=DISABLED"

printf -- '\n--- Updating agent stack'

parfait update-stack \
  -t templates/agent/template.yaml \
  "$AGENT_STACK" \
  "AgentDockerImage=${docker_image}" \
  "LambdaScheduleState=DISABLED"
