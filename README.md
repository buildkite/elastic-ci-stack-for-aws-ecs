# Elastic CI Stack for AWS: ECS Edition (2 elastic 2 stack)

This is an **experimental version** of our main [AWS stack](https://github.com/buildkite/elastic-ci-stack-for-aws) that makes use of ECS and Spot Fleets.

## Design Goals

 * Agents/Queues that each have their own IAM Role
 * Strong isolation for Agents/Queues
 * Shared underlying compute infrastructure via Spotfleet
 * Fast scaling, intervals of 10s

## Stacks

### VPC

The [VPC Stack](templates/vpc/README.md) provides an underlying VPC that will handle as many subnets as you have available.

### Spotfleet Compute

The [Spotfleet Stack](templates/compute/spotfleet/README.md) provides an ECS Cluster and an AWS Spotfleet that powers it. It auto-scales based on the needs of ECS Services in the Cluster.

### Agent

The [Agent Stack](templates/agent/README.md) provides an ECS Service that runs a Buildkite Agent as an ECS Task. Each Agent has it's own [IAM Task Role](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-iam-roles.html) which allows for strong isolation.

## Installation

Clone this repository and create each stack from the templates mentioned above.
