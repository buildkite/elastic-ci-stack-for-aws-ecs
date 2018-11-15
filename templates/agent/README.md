# Agent Stack

## Creating

```bash
# Set an agent registration token in SSM
aws ssm put-parameter --name "/buildkite/agent_token" --type String --value "xxx"

# Create the stack
aws cloudformation create-stack \
  --output text \
  --stack-name buildkite-agent \
  --template-body "file://$PWD/templates/agent/template.yaml" \
  --capabilities CAPABILITY_IAM \
  --parameters \
    "ParameterKey=LambdaBucket,ParameterValue=buildkite-aws-stack-ecs-dev"
```
