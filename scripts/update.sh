#!/bin/bash
set -euo pipefail

SPOT_FLEET_STACK=${SPOT_FLEET_STACK:-buildkite-spotfleet}
AGENT_STACK=${AGENT_STACK:-"buildkite-agent-default"}

echo '--- Updating spotfleet stack'
parfait update-stack \
  -t templates/compute/spotfleet/template.yaml \
  "${SPOT_FLEET_STACK}"

echo '--- Updating agent stack'
parfait update-stack \
  -t templates/agent/template.yaml \
  "${AGENT_STACK}"
