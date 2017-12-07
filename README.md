# Elastic CI Stack for AWS: ECS Edition (2 elastic 2 stack)

This is an **experimental version** of our main [AWS stack](https://github.com/buildkite/elastic-ci-stack-for-aws) that makes use of ECS and Spot Fleets.

The theory behind this is that there should be considerable cost savings.

## Design

* An AWS SpotFleet is used to run ECS Instances in a dedicated ECS cluster.
* An ECS Service is used to run our Agent via docker.
* A lambda ([ecs-agent-scaler](https://github.com/buildkite/buildkite-ecs-agent-scaler)) is run on a schedule to adjust the capacity of the SpotFleet and the number of agents in the Service
* ECS Instances bootstrap via user-data from a vanilla Amazon ECS AMI

## Open Questions

* Does bootstrapping vanilla ECS mean spin-up is slower?
* How does ECS scheduling handle builds that create docker containers outside of ECS?

