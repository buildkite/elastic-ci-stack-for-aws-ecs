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
    "ParameterKey=VPC,ParameterValue=${EC2_VPC_ID?}" \
    "ParameterKey=Subnets,ParameterValue=${EC2_VPC_SUBNETS//\\/}" \
    "ParameterKey=ECSCluster,ParameterValue=buildkite-agent"
```
