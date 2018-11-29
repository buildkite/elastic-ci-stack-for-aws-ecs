# Elastic CI Stack for AWS: ECS Edition (2 elastic 2 stack)

This is an **experimental version** of our main [AWS stack](https://github.com/buildkite/elastic-ci-stack-for-aws) that makes use of ECS and Spot Fleets.

## Design Goals

 * Agents/Queues that each have their own IAM Role
 * Strong isolation for Agents/Queues
 * Shared underlying compute infrastructure via Spotfleet
 * Fast auto-scaling

## How is isolation currently provided?

Agents are running in docker containers on ECS instances, each with their own [Task IAM Roles](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-iam-roles.html). The ECS Agent uses firewall rules to prevent containers from accessing the Instance Roles and also prevents usage of certain docker features like `privileged` and `host` networking.

A key part of lots of CI builds is being able to create docker containers and orchestrate them with docker-compose. This poses a serious challenge in that exposing the host docker socket to containers effectively bypasses all security and provides root access to the host. This stack uses [sockguard](https://github.com/buildkite/sockguard) to wrap the root docker socket and provide access-control and isolation.

## Caveats ☣️🚨🦑

* Agent session tokens (`BUILDKITE_AGENT_ACCESS_TOKEN`) are exposed to builds and are valid for the duration of the agent uptime. Exposing this token to third-party pull requests would be disasterous.
* To allow host binds of /buildkite/builds/xxx, the builds directory is currently stored on the ECS host and bind mounted into containers. This means all builds that run on a given ECS host can access each others checkout directories, including writing to them.
* [Sockguard](https://github.com/buildkite/sockguard) is very experimental and likely has critical bugs and security issues.
* Plugins that attempt to mount in /usr/local/bin/buildkite-agent to docker containers won't work, as it's not installed on the ECS hosts.

## Stacks

### VPC

The [VPC Stack](templates/vpc/README.md) provides an underlying VPC that will handle as many subnets as you have available.

### Spotfleet Compute

The [Spotfleet Stack](templates/compute/spotfleet/README.md) provides an ECS Cluster and an AWS Spotfleet that powers it. It auto-scales based on the needs of ECS Services in the Cluster.

### Agent

The [Agent Stack](templates/agent/README.md) provides an ECS Service that runs a Buildkite Agent as an ECS Task. Each Agent has it's own [Task IAM Roles](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-iam-roles.html), independent of the IAM permissions that the host that it's running on has.

## Installation

Clone this repository and create each stack from the templates mentioned above.
