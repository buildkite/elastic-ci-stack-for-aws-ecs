package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
)

const (
	version                = "0.0.0"
	defaultMetricsEndpoint = "https://agent.buildkite.com/v3"
	iterations             = 6
	delay                  = time.Second * 10
)

func main() {
	if os.Getenv(`DEBUG`) != "" {
		_, err := Handler(context.Background(), json.RawMessage([]byte{}))
		if err != nil {
			log.Fatal(err)
		}
	} else {
		lambda.Start(Handler)
	}
}

func Handler(ctx context.Context, evt json.RawMessage) (string, error) {
	for i := 0; i < iterations; i++ {
		client := newBuildkiteClient(os.Getenv(`BUILDKITE_TOKEN`))
		count, err := client.GetScheduledJobCount(os.Getenv(`BUILDKITE_QUEUE`))
		if err != nil {
			return "", err
		}

		cluster := os.Getenv(`BUILDKITE_ECS_CLUSTER`)
		service := os.Getenv(`BUILDKITE_ECS_SERVICE`)

		log.Printf("Modifying service %s, setting count=%d", service, count)

		svc := ecs.New(session.New())
		_, err = svc.UpdateService(&ecs.UpdateServiceInput{
			Cluster:      aws.String(cluster),
			Service:      aws.String(service),
			DesiredCount: aws.Int64(count),
		})
		if err != nil {
			return "", err
		}

		// Sleep so that we can get multiple executions in a single lambda run
		time.Sleep(delay)
	}

	return "", nil
}

type buildkiteClient struct {
	Endpoint   string
	AgentToken string
	UserAgent  string
	Queue      string
}

func newBuildkiteClient(agentToken string) *buildkiteClient {
	return &buildkiteClient{
		Endpoint:   defaultMetricsEndpoint,
		UserAgent:  fmt.Sprintf("elastic-ci-stack-for-aws/scaler/%s", version),
		AgentToken: agentToken,
	}
}

func (c *buildkiteClient) GetScheduledJobCount(queue string) (int64, error) {
	log.Printf("Collecting agent metrics for queue %q", queue)
	t := time.Now()

	endpoint, err := url.Parse(c.Endpoint)
	if err != nil {
		return 0, err
	}

	endpoint.Path += "/metrics"

	req, err := http.NewRequest("GET", endpoint.String(), nil)
	if err != nil {
		return 0, err
	}

	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Authorization", fmt.Sprintf("Token %s", c.AgentToken))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}

	var resp struct {
		Jobs struct {
			Queues map[string]struct {
				Total int64 `json:"total"`
			} `json:"queues"`
		} `json:"jobs"`
	}

	defer res.Body.Close()
	err = json.NewDecoder(res.Body).Decode(&resp)
	if err != nil {
		return 0, err
	}

	var count int64

	if queue, exists := resp.Jobs.Queues[queue]; exists {
		count = queue.Total
	}

	log.Printf("↳ Got %d total jobs (took %v)", count, time.Now().Sub(t))
	return count, nil
}