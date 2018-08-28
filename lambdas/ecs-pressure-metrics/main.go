package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/ecs"
)

const (
	version = "0.0.0"
)

const (
	iterations = 6
	delay      = time.Second * 10
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
		clusterName := os.Getenv(`BUILDKITE_ECS_CLUSTER`)

		sess := session.New()
		svc := ecs.New(sess)

		listServicesOutput, err := svc.ListServices(&ecs.ListServicesInput{
			Cluster: aws.String(clusterName),
		})
		if err != nil {
			return "", err
		}

		describeServicesOutput, err := svc.DescribeServices(&ecs.DescribeServicesInput{
			Cluster:  aws.String(clusterName),
			Services: listServicesOutput.ServiceArns,
		})
		if err != nil {
			return "", err
		}

		var cpuRequired int64
		var memoryRequired int64

		if svcLen := len(describeServicesOutput.Services); svcLen == 0 {
			log.Printf("No services defined")
			return "", nil
		}

		for _, service := range describeServicesOutput.Services {
			log.Printf("Service %s has desired=%d, running=%d, pending=%d",
				*service.ServiceName, *service.DesiredCount, *service.RunningCount, *service.PendingCount)

			describeTaskDefinitionResult, err := svc.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
				TaskDefinition: service.TaskDefinition,
			})
			if err != nil {
				return "", err
			}

			taskCPURequired, err := strconv.ParseInt(*describeTaskDefinitionResult.TaskDefinition.Cpu, 10, 64)
			if err != nil {
				return "", err
			}

			cpuRequired += (taskCPURequired * *service.DesiredCount)

			taskMemoryRequired, err := strconv.ParseInt(*describeTaskDefinitionResult.TaskDefinition.Memory, 10, 64)
			if err != nil {
				return "", err
			}

			memoryRequired += (taskMemoryRequired * *service.DesiredCount)
		}

		resources, err := getContainerInstanceResources(sess, clusterName)
		if err != nil {
			return "", err
		}

		log.Printf("Cluster %s has %d instances with %d CPU shares and %dMB of memory",
			clusterName, resources.Count, resources.CPU, resources.Memory)

		cpuPressure := cpuRequired - resources.CPU
		memoryPressure := memoryRequired - resources.Memory

		log.Printf("Buildkite/ECS/CPUPressure: %d", cpuPressure)
		log.Printf("Buildkite/ECS/MemoryPressure: %d", memoryPressure)

		// Send metrics to Cloudwatch Buildkite/ECS
		_, err = cloudwatch.New(sess).PutMetricData(&cloudwatch.PutMetricDataInput{
			MetricData: []*cloudwatch.MetricDatum{
				&cloudwatch.MetricDatum{
					MetricName: aws.String(`CPUPressure`),
					Dimensions: []*cloudwatch.Dimension{
						{Name: aws.String(`Cluster`), Value: aws.String(clusterName)},
					},
					Value: aws.Float64(float64(cpuPressure)),
					Unit:  aws.String("Count"),
				},
				&cloudwatch.MetricDatum{
					MetricName: aws.String(`MemoryPressure`),
					Dimensions: []*cloudwatch.Dimension{
						{Name: aws.String(`Cluster`), Value: aws.String(clusterName)},
					},
					Value: aws.Float64(float64(memoryPressure)),
					Unit:  aws.String("Megabytes"),
				},
			},
			Namespace: aws.String(`Buildkite/ECS`),
		})
		if err != nil {
			return "", err
		}

		log.Printf("Published cloudwatch metrics")

		// Sleep so that we can get multiple executions in a single lambda run
		time.Sleep(delay)
	}

	return "", nil
}

type containerInstanceResources struct {
	CPU    int64
	Memory int64
	Count  int
}

func getContainerInstanceResources(sess *session.Session, cluster string) (res containerInstanceResources, err error) {
	svc := ecs.New(sess)

	listResult, err := svc.ListContainerInstances(&ecs.ListContainerInstancesInput{
		Cluster: aws.String(cluster),
	})
	if err != nil {
		return res, err
	}

	// no container instances
	if len(listResult.ContainerInstanceArns) == 0 {
		return containerInstanceResources{}, nil
	}

	result, err := svc.DescribeContainerInstances(&ecs.DescribeContainerInstancesInput{
		Cluster:            aws.String(cluster),
		ContainerInstances: listResult.ContainerInstanceArns,
	})
	if err != nil {
		return res, err
	}

	for _, instance := range result.ContainerInstances {
		res.Count++
		for _, resource := range instance.RemainingResources {
			switch *resource.Name {
			case "CPU":
				res.CPU += *resource.IntegerValue
			case "MEMORY":
				res.Memory += *resource.IntegerValue
			}
		}
	}

	return res, nil
}
