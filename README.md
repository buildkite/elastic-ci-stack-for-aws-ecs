# Elastic CI Stack for AWS: ECS Edition (2 elastic 2 stack)

This is an **experimental version** of our main [AWS stack](https://github.com/buildkite/elastic-ci-stack-for-aws) that makes use of ECS and Spot Fleets.

## Stacks

### VPC

The [VPC Stack](templates/vpc/README.md) provides the underlying VPC resources.

### Agent

The [Agent Stack](templates/agent/README.md) provides the top-level ECS Cluster, with the buildkite-agent ECS Task and Service. It has two lambdas that auto-scale the ECS service based on Scheduled Jobs in the Buildkite API and also publishes metrics on the compute capacity required by the ECS service.

### Spotfleet Compute

The [Spotfleet Stack](templates/compute/spotfleet/README.md) provides an AWS Spotfleet that runs ECS Instances. It autoscales based on metrics published by the Metrics stack.
