#!/bin/sh
set -euo pipefail

# Detect the container ID from /proc/self/cgroup
container_id=$(cat /proc/self/cgroup | grep "1:name=systemd" | rev | cut -d/ -f1 | rev)
echo "Container ID is $container_id"

# Get the CgroupParent via the Docker API
container_inspect_url="http:/v1.37/containers/${container_id}/json"
cgroup_parent=$(curl -s --unix-socket /var/run/docker.sock "$container_inspect_url" | jq -r .HostConfig.CgroupParent)

if [ -z "$cgroup_parent" ]; then
  echo "cgroup_parent empty? (from Docker API)"
  exit 1
fi

exec /usr/bin/sockguard \
  -filename /var/run/sockguard/docker.sock \
  -allow-bind /buildkite/builds \
  -cgroup-parent "${cgroup_parent}" \
  -owner-label "${cgroup_parent}"
