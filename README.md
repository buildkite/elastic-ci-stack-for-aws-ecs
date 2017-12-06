# Elastic CI Stack for AWS: ECS Edition

This is an experimental version of our main AWS stack that makes use of ECS and Spot Fleets.

## Design

An AWS SpotFleet is used to run ECS Instances in a dedicated ECS cluster. There is service that runs Agents on specific hosts.
