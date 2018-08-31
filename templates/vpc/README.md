# Elastic Stack VPC Template

The VPC stack creates a VPC with as many availabilty zones as are available in your region to optimize spot pricing. It makes the following configurations available:

* 2 public subnets
* 3 public subnets
* 4 public subnets
* 5 public subnets
* 6 public subnets
* 2 private subnets + 2 public subnets with NAT Gateways
* 3 private subnets + 2 public subnets with NAT Gateways
* 4 private subnets + 2 public subnets with NAT Gateways
* 5 private subnets + 2 public subnets with NAT Gateways
* 6 private subnets + 2 public subnets with NAT Gateways

Note that NAT Gateways are kind of expensive and are charged at $0.045-0.095 an hour.

## Creating

```bash
## Figure out what availability zones are available
aws ec2 describe-availability-zones \
  --query 'AvailabilityZones[?State==`available`].ZoneName'
[
    "us-east-1a",
    "us-east-1b",
    "us-east-1c",
    "us-east-1d",
    "us-east-1e",
    "us-east-1f"
]

## Create a VPC stack
aws cloudformation create-stack \
  --output text \
  --stack-name elastic-ecs-vpc \
  --template-body "file://$PWD/templates/vpc/template.yaml" \
  --parameters \
    "ParameterKey=AvailabilityZones,ParameterValue=us-east-1a\\,us-east-1b\\,us-east-1c\\,us-east-1d\\,us-east-1e\\,us-east-1f" \
    "ParameterKey=SubnetConfiguration,ParameterValue=6 public subnets"

## Get Private Subnets and Vpc from VPC stack
aws cloudformation describe-stacks \
  --stack-name elastic-ecs-vpc \
  --query "Stacks[*].Outputs"
```
