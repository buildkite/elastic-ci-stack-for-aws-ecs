.PHONY: lint validate docker

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
	zip -j $< $@

lambdas/ecs-service-scaler/handler: lambdas/ecs-service-scaler/main.go
	docker run --volume $(PWD):/go/src/github.com/buildkite/elastic-stack-for-aws-ecs \
		--workdir /go/src/github.com/buildkite/elastic-stack-for-aws-ecs \
		--rm golang:1.10 \
		sh -c "go get ./lambdas/ecs-service-scaler && \
			go build -o lambdas/ecs-service-scaler/handler ./lambdas/ecs-service-scaler"
	chmod +x lambdas/ecs-service-scaler/handler

lambdas/ecs-spotfleet-scaler/handler: lambdas/ecs-spotfleet-scaler/main.go
	docker run --volume $(PWD):/go/src/github.com/buildkite/elastic-stack-for-aws-ecs \
		--workdir /go/src/github.com/buildkite/elastic-stack-for-aws-ecs \
		--rm golang:1.10 \
		sh -c "go get ./lambdas/ecs-spotfleet-scaler && \
			go build -o lambdas/ecs-spotfleet-scaler/handler ./lambdas/ecs-spotfleet-scaler"
	chmod +x lambdas/ecs-spotfleet-scaler/handler

sync: $(LAMBDAS)
	aws s3 cp --acl public-read ecs-service-scaler.zip s3://$(LAMBDA_S3_BUCKET)$(LAMBDA_S3_BUCKET_PATH)
	aws s3 cp --acl public-read ecs-spotfleet-scaler.zip s3://$(LAMBDA_S3_BUCKET)$(LAMBDA_S3_BUCKET_PATH)

docker:
	docker pull buildkite/agent:3
	docker build --tag "$(DOCKER_TAG)" ./docker

lint:
	find templates -name '*.yaml' | xargs -n1 cfn-lint
