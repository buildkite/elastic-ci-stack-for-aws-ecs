# Spotfleet Compute Stack

## Creating

```bash
# Create the stack
aws cloudformation create-stack \
  --output text \
  --stack-name buildkite-spotfleet \
  --template-body "file://$PWD/templates/compute/spotfleet/template.yaml" \
  --capabilities CAPABILITY_IAM \
  --parameters \
    "ParameterKey=VPC,ParameterValue=vpc-056ebcd495d88a4e5" \
    "ParameterKey=Subnets,ParameterValue=subnet-068847849d08cbe07\\,subnet-0a1064ff38b7a65c5\\,subnet-0b3c85a9218691cc8\\,subnet-0b35721bba36a3e71\\,subnet-09d817e59eb4b8f64\\,subnet-0f330114d9952c36f" \
    "ParameterKey=ECSCluster,ParameterValue=buildkite-agent"
```
