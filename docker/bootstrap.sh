#!/bin/bash
set -euo pipefail

if [[ -z "${USE_ECS_BOOTSTRAP:-}" ]] ; then
  exec buildkite-agent bootstrap
fi

task_family="buildkite_${BUILDKITE_ORGANIZATION_SLUG//-/_}_${BUILDKITE_PIPELINE_SLUG//-/_}"
task_definition=$(cat <<EOF
{
  "containerDefinitions": [
    {
      "essential": true,
      "image": "${BUILDKITE_BOOTSTRAP_DOCKER_IMAGE}",
      "memory": 100,
      "name": "${task_family}"
    }
  ],
  "family": "${task_family}"
}
EOF
)

exec ecs-run-task \
  --cluster "${ECS_CLUSTER}" \
  --log-group "$BUILDKITE_ECS_RUN_LOG_GROUP" \
  --file <(echo "$task_definition") \
  --inherit-env \
  bootstrap
