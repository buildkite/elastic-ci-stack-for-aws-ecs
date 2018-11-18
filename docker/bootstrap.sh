#!/bin/bash
set -euo pipefail

if [[ "$OSTYPE" =~ ^(darwin|win32) ]] ; then
  echo "Docker bootstrap only works on linux at present!"
  exit 1
fi

DOCKER_IMAGE="buildkite/agent:latest"
DOCKER_SOCKET_PATH="/var/run/docker.sock"
EXPOSE_DOCKER_SOCKET=false
DOCKER_BUILD_VOLUME="${BUILDKITE_ORGANIZATION_SLUG}_${BUILDKITE_PIPELINE_SLUG}"

# Build an array of params to pass to docker run
args=(
  --env BUILDKITE_AGENT_ACCESS_TOKEN
  --env "BUILDKITE_BUILD_PATH=$BUILDKITE_BUILD_PATH"
  --volume "${DOCKER_BUILD_VOLUME}:$BUILDKITE_BUILD_PATH"
)

# This trick ensures runs the docker container as unprivileged, but matches userids on the host
if [[ "${USER:-root}" != "root" ]] ; then
  args+=(
    --volume /etc/group:/etc/group:ro
    --volume /etc/passwd:/etc/passwd:ro
    --user "$( id -u "$USER" ):$( id -g "$USER" )"
    "--security-opt=no-new-privileges"
  )
fi

# Optionally expose the docker socket for builds
if [[ "$EXPOSE_DOCKER_SOCKET" =~ ^(true|1|on)$ ]] ; then
  args+=(--volume "${DOCKER_SOCKET_PATH}:/var/run/docker.sock")
fi

# Read in the env file and convert to --env params for docker
while read -r var; do
  args+=( --env "${var%%=*}" )
done < "$BUILDKITE_ENV_FILE"

# Invoke the bootstrap in a docker container
echo "~~~ Build running in :docker: ${DOCKER_IMAGE}"
docker run "${args[@]}" "$DOCKER_IMAGE" bootstrap "$@"
