.PHONY: sync lint validate docker
VERSION=$(shell git describe --tags --candidates=1 --dirty 2>/dev/null \
	|| printf "dev-%s" "$$(git rev-parse --short HEAD)")
FLAGS=-s -w -X main.Version=$(VERSION)

LAMBDA_S3_BUCKET := buildkite-aws-stack-ecs-dev
LAMBDA_S3_BUCKET_PATH := /
DOCKER_TAG := buildkite/agent-ecs

LAMBDAS = ecs-service-scaler.zip ecs-spotfleet-scaler.zip

build: $(LAMBDAS)

clean:
	-rm $(LAMBDAS)
	-rm lambdas/ecs-service-scaler/handler
	-rm lambdas/ecs-spotfleet-scaler/handler

%.zip: lambdas/%/handler
	zip -9 -v -j $@ -i "$<"

lambdas/ecs-service-scaler/handler: lambdas/ecs-service-scaler/main.go lambdas/ecs-service-scaler/go.sum
	docker run \
		--volume module_cache:/go/pkg/mod \
		--volume $(PWD)/lambdas/ecs-service-scaler:/lambda \
		--workdir /lambda \
		--rm golang:1.11 \
		go build -ldflags="$(FLAGS)" -o handler .
	chmod +x lambdas/ecs-service-scaler/handler

lambdas/ecs-spotfleet-scaler/handler: lambdas/ecs-spotfleet-scaler/main.go lambdas/ecs-spotfleet-scaler/go.sum
	docker run \
		--volume module_cache:/go/pkg/mod \
		--volume $(PWD)/lambdas/ecs-spotfleet-scaler:/lambda \
		--workdir /lambda \
		--rm golang:1.11 \
		go build -ldflags="$(FLAGS)" -o handler .
	chmod +x lambdas/ecs-spotfleet-scaler/handler

sync: $(LAMBDAS)
	aws s3 cp --acl public-read ecs-service-scaler.zip s3://$(LAMBDA_S3_BUCKET)$(LAMBDA_S3_BUCKET_PATH)
	aws s3 cp --acl public-read ecs-spotfleet-scaler.zip s3://$(LAMBDA_S3_BUCKET)$(LAMBDA_S3_BUCKET_PATH)

docker:
	docker build --tag "$(DOCKER_TAG)" ./docker

lint:
	find templates -name '*.yaml' | xargs -n1 cfn-lint
