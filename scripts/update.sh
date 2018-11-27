#!/bin/bash
set -euo pipefail

STACK_SUFFIX=dev

echo '--- Updating spotfleet stack'
parfait update-stack \
  -t templates/compute/spotfleet/template.yaml \
  buildkite-spotfleet-${STACK_SUFFIX}

echo '--- Updating agent stack'
parfait update-stack \
  -t templates/agent/template.yaml \
  buildkite-agent-${STACK_SUFFIX}
