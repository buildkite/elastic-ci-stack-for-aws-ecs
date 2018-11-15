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
export EC2_AVAILABILITY_ZONES=$(aws ec2 describe-availability-zones \
  --query 'AvailabilityZones[?State==`available`].ZoneName' \
  --output text | sed -E -e 's/[[:blank:]]+/,/g')

export EC2_AVAILABILITY_ZONES_COUNT=$(grep -c ',' <<< "$EC2_AVAILABILITY_ZONES")

## Create a VPC stack
aws cloudformation create-stack \
  --output text \
  --stack-name elastic-ecs-vpc \
  --template-body "file://$PWD/templates/vpc/template.yaml" \
  --parameters \
    "ParameterKey=AvailabilityZones,ParameterValue=${EC2_AVAILABILITY_ZONES}" \
    "ParameterKey=SubnetConfiguration,ParameterValue=${EC2_AVAILABILITY_ZONES_COUNT} public subnets"

## Get Private Subnets and Vpc from VPC stack
export EC2_VPC_ID="$(aws cloudformation describe-stacks \
  --stack-name elastic-ecs-vpc \
  --query 'Stacks[0].Outputs[?OutputKey==`Vpc`].OutputValue' \
  --output text)"

export EC2_VPC_SUBNETS="$(aws cloudformation describe-stacks \
  --stack-name elastic-ecs-vpc \
  --query 'Stacks[0].Outputs[?OutputKey==`PublicSubnets`].OutputValue' \
  --output text | sed -e 's/,/\\\\,/g')"
```
