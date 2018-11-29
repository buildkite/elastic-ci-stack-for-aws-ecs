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
	conf := config{
		BuildkiteToken: os.Getenv(`BUILDKITE_TOKEN`),
		BuildkiteQueue: os.Getenv(`BUILDKITE_QUEUE`),
		ECSCluster: os.Getenv(`ECS_CLUSTER`),
		ECSService: os.Getenv(`ECS_SERVICE`),
	}

	for {
		select {
		case <-timeout:
			return "", nil
		default:
			err := scaleECSServiceCapacity(sess, conf)
			if err != nil {
				log.Printf("Err: %#v", err.Error())
				return "", nil
			}

			log.Printf("Sleeping for %v", interval)
			time.Sleep(interval)
		}
	}
}

type config struct {
	BuildkiteToken string
	BuildkiteQueue string
	ECSCluster string
	ECSService string
}

func scaleECSServiceCapacity(sess *session.Session, config config) error {
	client := newBuildkiteClient(config.BuildkiteToken)
	count, err := client.GetScheduledJobCount(config.BuildkiteQueue)
	if err != nil {
		return err
	}

	svc := ecs.New(sess)

	result, err := svc.DescribeServices(&ecs.DescribeServicesInput{
		Cluster:      aws.String(config.ECSCluster),
		Services: []*string{
			aws.String(config.ECSService),
		},
	})
	if err != nil {
		return err
	}

	if len(result.Services[0].Deployments) > 1 {
		log.Printf("Deployment in progress, waiting")
		return nil
	}

	log.Printf("Modifying service %s, setting count=%d", config.ECSService, count)
	_, err = svc.UpdateService(&ecs.UpdateServiceInput{
		Cluster:      aws.String(config.ECSCluster),
		Service:      aws.String(config.ECSService),
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
