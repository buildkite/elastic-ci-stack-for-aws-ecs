package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
)

const (
	// Converts cpu/memory needed into a capacity figure for spotfleet
	cpuDivisor    = 1024
	memoryDivisor = 2048
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

	clusterName := os.Getenv(`BUILDKITE_ECS_CLUSTER`)
	spotFleet := os.Getenv(`BUILDKITE_SPOT_FLEET`)
	sess := session.New()

	for {
		select {
		case <-timeout:
			return "", nil
		default:
			err := scaleSpotFleetCapacity(sess, clusterName, spotFleet)
			if err != nil {
				log.Printf("Err: %#v", err.Error())
				return "", nil
			}

			log.Printf("Sleeping for %v", interval)
			time.Sleep(interval)
		}
	}
}

func scaleSpotFleetCapacity(sess *session.Session, clusterName, spotFleet string) error {
	svc := ecs.New(sess)
	listServicesOutput, err := svc.ListServices(&ecs.ListServicesInput{
		Cluster: aws.String(clusterName),
	})
	if err != nil {
		return err
	}

	describeServicesOutput, err := svc.DescribeServices(&ecs.DescribeServicesInput{
		Cluster:  aws.String(clusterName),
		Services: listServicesOutput.ServiceArns,
	})
	if err != nil {
		return err
	}

	var cpuRequired int64
	var memoryRequired int64

	if svcLen := len(describeServicesOutput.Services); svcLen == 0 {
		log.Printf("No services defined")
		return nil
	}

	for _, service := range describeServicesOutput.Services {
		log.Printf("Service %s has desired=%d, running=%d, pending=%d",
			*service.ServiceName, *service.DesiredCount, *service.RunningCount, *service.PendingCount)

		describeTaskDefinitionResult, err := svc.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
			TaskDefinition: service.TaskDefinition,
		})
		if err != nil {
			return err
		}

		taskCPURequired, err := strconv.ParseInt(*describeTaskDefinitionResult.TaskDefinition.Cpu, 10, 64)
		if err != nil {
			return err
		}

		cpuRequired += (taskCPURequired * *service.DesiredCount)

		taskMemoryRequired, err := strconv.ParseInt(*describeTaskDefinitionResult.TaskDefinition.Memory, 10, 64)
		if err != nil {
			return err
		}

		memoryRequired += (taskMemoryRequired * *service.DesiredCount)
	}

	log.Printf("Total needed CPU is %d, total needed memory is %d", cpuRequired, memoryRequired)

	var count int64 = int64(cpuRequired / cpuDivisor)
	if int64(memoryRequired/memoryDivisor) > count {
		count = int64(memoryRequired / memoryDivisor)
	}

	ec2Svc := ec2.New(sess)

	describeSpotFleetOutput, err := ec2Svc.DescribeSpotFleetRequests(&ec2.DescribeSpotFleetRequestsInput{
		SpotFleetRequestIds: []*string{
			aws.String(spotFleet),
		},
	})
	if err != nil {
		return err
	}

	if len(describeSpotFleetOutput.SpotFleetRequestConfigs) == 0 {
		return fmt.Errorf("No spot fleet found for %s", spotFleet)
	}

	spotFleetConfig := describeSpotFleetOutput.SpotFleetRequestConfigs[0]

	log.Printf("Spotfleet %s has target=%d",
		spotFleet,
		*spotFleetConfig.SpotFleetRequestConfig.TargetCapacity,
	)

	// Spot fleet can't be modified whilst in "modifying"
	if *spotFleetConfig.SpotFleetRequestState == "modifying" {
		log.Printf("Spot fleet is presently in %q state", *spotFleetConfig.SpotFleetRequestState)
		return nil
	}

	// Don't change spot fleet if it's already at TargetCapacity
	if *spotFleetConfig.SpotFleetRequestConfig.TargetCapacity == count {
		log.Printf("TargetCapacity is already at correct count")
		return nil
	}

	log.Printf("Modifying spotfleet %s, setting TargetCapacity=%d", spotFleet, count)

	_, err = ec2Svc.ModifySpotFleetRequest(&ec2.ModifySpotFleetRequestInput{
		SpotFleetRequestId: aws.String(spotFleet),
		TargetCapacity:     aws.Int64(count),
	})
	if err != nil {
		return err
	}

	return nil
}
