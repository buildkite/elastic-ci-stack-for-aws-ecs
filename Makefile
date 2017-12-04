
all: build sync

build:
	go run generate.go

sync:
	aws s3 sync --acl public-read templates/ s3://buildkite-aws-stack-ecs-dev
