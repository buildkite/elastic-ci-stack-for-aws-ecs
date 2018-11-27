package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
)

const (
	defaultMetricsEndpoint = "https://agent.buildkite.com/v3"
)

var (
	Version string = "dev"
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
	log.Printf("ecs-spotfleet-scaler version %s", Version)

	var timeout <-chan time.Time = make(chan time.Time)
	var interval time.Duration = 10 * time.Second

	if intervalStr := os.Getenv(`LAMBDA_INTERVAL`); intervalStr != "" {
		var err error
		interval, err = time.ParseDuration(intervalStr)
		if err != nil {
			return "", err
		}
	}

	if timeoutStr := os.Getenv(`LAMBDA_TIMEOUT`); timeoutStr != "" {
		timeoutDuration, err := time.ParseDuration(timeoutStr)
		if err != nil {
			return "", err
		}
		timeout = time.After(timeoutDuration)
	}

	sess := session.New()

	for {
		select {
		case <-timeout:
			return "", nil
		default:
			err := scaleECSServiceCapacity(sess)
			if err != nil {
				log.Printf("Err: %#v", err.Error())
				return "", nil
			}

			log.Printf("Sleeping for %v", interval)
			time.Sleep(interval)
		}
	}
}

func scaleECSServiceCapacity(sess *session.Session) error {
	client := newBuildkiteClient(os.Getenv(`BUILDKITE_TOKEN`))
	count, err := client.GetScheduledJobCount(os.Getenv(`BUILDKITE_QUEUE`))
	if err != nil {
		return err
	}

	cluster := os.Getenv(`BUILDKITE_ECS_CLUSTER`)
	service := os.Getenv(`BUILDKITE_ECS_SERVICE`)

	var minSize int64
	if ms := os.Getenv(`BUILDKITE_MIN_SIZE`); ms != "" {
		var err error
		minSize, err = strconv.ParseInt(ms, 10, 32)
		if err != nil {
			return fmt.Errorf("failed to parse BUILDKITE_MIN_SIZE: %v", err)
		}
	}

	svc := ecs.New(sess)

	result, err := svc.DescribeServices(&ecs.DescribeServicesInput{
		Cluster:      aws.String(cluster),
		Services: []*string{
			aws.String(service),
		},
	})
	if err != nil {
		return err
	}

	if len(result.Services[0].Deployments) > 1 {
		log.Printf("Deployment in progress, waiting")
		return nil
	}

	if count < minSize {
		log.Printf("Adjusting count to maintain minimum size of %d, would have been %d", 
			minSize, count)
		count = minSize
	}

	log.Printf("Modifying service %s, setting count=%d", service, count)
	_, err = svc.UpdateService(&ecs.UpdateServiceInput{
		Cluster:      aws.String(cluster),
		Service:      aws.String(service),
		DesiredCount: aws.Int64(count),
	})
	if err != nil {
		return  err
	}

	return nil
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
		UserAgent:  fmt.Sprintf("elastic-ci-stack-for-aws-ecs/ecs-service-scaler/%s", Version),
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
