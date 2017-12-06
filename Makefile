.PHONY: all clean build sync lint

all: build sync

build:
	go run generate.go

sync:
	aws s3 sync --acl public-read templates/ s3://buildkite-aws-stack-ecs-dev

lint:
	find templates -name '*.yaml' | xargs -n1 cfn-lint validate
