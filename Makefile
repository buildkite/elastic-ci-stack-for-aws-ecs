.PHONY: sync lint validate

sync:
	aws s3 sync --acl public-read templates/ s3://buildkite-aws-stack-ecs-dev

lint:
	find templates -name '*.yaml' | xargs -n1 cfn-lint

validate:
	aws cloudformation validate-template --template-body file://templates/vpc.yaml
	aws cloudformation validate-template --template-body file://templates/spotfleet.yaml
	aws cloudformation validate-template --template-body file://templates/elastic-stack.yaml
