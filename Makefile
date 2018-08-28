.PHONY: lint validate

LAMBDA_S3_BUCKET := buildkite-aws-stack-ecs-dev
LAMBDA_S3_BUCKET_PATH := /

build: ecs-service-scaler.zip

clean:
	-rm ecs-service-scaler.zip
	-rm lambdas/ecs-service-scaler/handler

ecs-pressure-metrics.zip: lambdas/ecs-pressure-metrics/handler
	zip -j ecs-pressure-metrics.zip lambdas/ecs-pressure-metrics/handler

ecs-service-scaler.zip: lambdas/ecs-service-scaler/handler
	zip -j ecs-service-scaler.zip lambdas/ecs-service-scaler/handler

lambdas/ecs-service-scaler/handler: lambdas/ecs-service-scaler/main.go
	docker run --volume $(PWD):/go/src/github.com/buildkite/elastic-stack-for-aws-ecs \
		--workdir /go/src/github.com/buildkite/elastic-stack-for-aws-ecs \
		--rm golang:1.10 \
		sh -c "go get ./lambdas/ecs-service-scaler && go build -o lambdas/ecs-service-scaler/handler ./lambdas/ecs-service-scaler"
	chmod +x lambdas/ecs-service-scaler/handler

lambdas/ecs-pressure-metrics/handler: lambdas/ecs-pressure-metrics/main.go
	docker run --volume $(PWD):/go/src/github.com/buildkite/elastic-stack-for-aws-ecs \
		--workdir /go/src/github.com/buildkite/elastic-stack-for-aws-ecs \
		--rm golang:1.10 \
		sh -c "go get ./lambdas/ecs-pressure-metrics && go build -o lambdas/ecs-pressure-metrics/handler ./lambdas/ecs-pressure-metrics"
	chmod +x lambdas/ecs-pressure-metrics/handler

sync: ecs-service-scaler.zip ecs-pressure-metrics.zip
	aws s3 cp --acl public-read ecs-service-scaler.zip s3://$(LAMBDA_S3_BUCKET)$(LAMBDA_S3_BUCKET_PATH)
	aws s3 cp --acl public-read ecs-pressure-metrics.zip s3://$(LAMBDA_S3_BUCKET)$(LAMBDA_S3_BUCKET_PATH)

lint:
	find templates -name '*.yaml' | xargs -n1 cfn-lint
