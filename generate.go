package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
)

const yamlTemplate = `
Description: >
  Spot Fleet for ECS

Parameters:
  VPC:
    Description: The VPC to deploy SpotFleet into
    Type: AWS::EC2::VPC::Id

  Subnets:
    Description: The subnets of a VPC to deploy SpotFleet into
    Type: List<AWS::EC2::Subnet::Id>

  ECSCluster:
    Description: The ECS Cluster to register with
    Type: String

  AMI:
    Description: The AMI to launch instances with
    Type: String

  InitialCapacity:
    Description: The initial capacity for the spot fleet
    Type: Number

  MaxPricePerUnitHour:
    Description: The maximum to bid per unit-hour
    Type: String

Resources:
  ECSSecurityGroup:
    Type: AWS::EC2::SecurityGroup
    Properties:
      GroupName: "Security group for ECS instances"
      GroupDescription: "Security group for ECS instances"
      VpcId: !Ref VPC

  ECSSecurityGroupIngress:
    Type: AWS::EC2::SecurityGroupIngress
    Properties:
      GroupId: !Ref ECSSecurityGroup
      SourceSecurityGroupId: !Ref ECSSecurityGroup
      IpProtocol: tcp
      FromPort: 0
      ToPort: 65535

  SpotFleetIAMRole:
    Type: AWS::IAM::Role
    Properties:
      AssumeRolePolicyDocument: |
        {
          "Statement": [{
            "Action": "sts:AssumeRole",
            "Effect": "Allow",
            "Principal": {
              "Service": "spotfleet.amazonaws.com"
            }
          }]
        }
      Path: /
      ManagedPolicyArns:
        - arn:aws:iam::aws:policy/service-role/AmazonEC2SpotFleetRole

  ECSRole:
    Type: AWS::IAM::Role
    Properties:
      Path: /
      RoleName: !Sub ${AWS::StackName}-ECSRole-${AWS::Region}
      AssumeRolePolicyDocument: |
        {
          "Statement": [{
            "Action": "sts:AssumeRole",
            "Effect": "Allow",
            "Principal": {
              "Service": "ec2.amazonaws.com"
            }
          }]
        }
      Policies:
        - PolicyName: ecs-service
          PolicyDocument: |
            {
              "Statement": [{
                "Effect": "Allow",
                "Action": [
                "ecs:CreateCluster",
                "ecs:DeregisterContainerInstance",
                "ecs:DiscoverPollEndpoint",
                "ecs:Poll",
                "ecs:RegisterContainerInstance",
                "ecs:StartTelemetrySession",
                "ecs:Submit*",
                "logs:CreateLogStream",
                "logs:PutLogEvents",
                "ecr:BatchCheckLayerAvailability",
                "ecr:BatchGetImage",
                "ecr:GetDownloadUrlForLayer",
                "ecr:GetAuthorizationToken"
                ],
                "Resource": "*"
              }]
            }

  ECSInstanceProfile:
    Type: AWS::IAM::InstanceProfile
    Properties:
      Path: /
      Roles:
        - !Ref ECSRole

  SpotFleet:
    Type: AWS::EC2::SpotFleet
    Properties:
      SpotFleetRequestConfigData:
        AllocationStrategy: lowestPrice
        IamFleetRole: !GetAtt SpotFleetIAMRole.Arn
		TargetCapacity: !Ref InitialCapacity
		SpotPrice: !Ref MaxPricePerUnitHour
        LaunchSpecifications: {{range $spec := . }}
          - WeightedCapacity: {{ $spec.Weight }}
            IamInstanceProfile: { Arn: !GetAtt ECSInstanceProfile.Arn }
            InstanceType: {{ $spec.InstanceType }}
            ImageId: !Ref AMI
            NetworkInterfaces:
              - DeviceIndex: 0
                SubnetId: !Select [ {{ .SubnetOffset }}, !Ref Subnets ]
                Groups: [ !GetAtt ECSSecurityGroup.GroupId ]
            UserData:
              "Fn::Base64": !Sub |
                #!/bin/bash
                yum install -y aws-cfn-bootstrap
                echo ECS_CLUSTER=${ECSCluster} >> /etc/ecs/ecs.config
                /opt/aws/bin/cfn-signal -e $? --stack ${AWS::StackName} --resource SpotFleet --region ${AWS::Region}
        {{end}}

Outputs:
  SpotFleet:
    Description: The SpotFleet created
    Value: !Ref SpotFleet
`

const (
	specsPerTemplate = 50
	maxWeight        = 16
)

// We can only have 50 specifications in total. So we have sizes * availability zones. We choose different sets for
// our different combinations to maximize our selections
// from https://aws.amazon.com/ec2/spot/pricing/
// and https://docs.aws.amazon.com/general/latest/gr/rande.html#ec2_region

var allSpecs = []Specification{
	{"m5.large", 4}, {"m5.xlarge", 8}, {"m5.2xlarge", 16}, {"m5.4xlarge", 32}, {"m5.12xlarge", 96}, {"m5.24xlarge", 192},
	{"c5.large", 4}, {"c5.xlarge", 8}, {"c5.2xlarge", 16}, {"c5.4xlarge", 32}, {"c5.9xlarge", 72}, {"c5.9xlarge", 144},
	{"t2.small", 1}, {"t2.medium", 2}, {"t2.large", 4}, {"t2.xlarge", 8},
	{"m3.medium", 2}, {"m3.large", 4}, {"m3.xlarge", 8}, {"m3.2xlarge", 16},
	{"m4.large", 4}, {"m4.xlarge", 8}, {"m4.2xlarge", 16}, {"m4.4xlarge", 32}, {"m4.10xlarge", 80}, {"m4.16xlarge", 128},
	{"c4.large", 4}, {"c4.xlarge", 8}, {"c4.2xlarge", 16}, {"c4.4xlarge", 32}, {"c4.8xlarge", 64},
	{"c3.large", 4}, {"c3.xlarge", 8}, {"c3.2xlarge", 16}, {"c3.4xlarge", 32}, {"c3.8xlarge", 64},
	{"i2.large", 4}, {"i2.xlarge", 8}, {"i2.2xlarge", 16}, {"i2.4xlarge", 32}, {"i2.8xlarge", 64},
	{"i3.large", 4}, {"i3.xlarge", 8}, {"i3.2xlarge", 16}, {"i3.4xlarge", 32}, {"i3.8xlarge", 64}, {"i3.16xlarge", 128},
}

var regions = []Region{
	{Name: "us-east-2", Zones: 3, Specifications: []Specification{
		{"t2.small", 1}, {"t2.medium", 2}, {"t2.large", 4}, {"t2.xlarge", 8},
		{"m4.large", 4}, {"m4.xlarge", 8}, {"m4.2xlarge", 16}, {"m4.4xlarge", 32}, {"m4.10xlarge", 80}, {"m4.16xlarge", 128},
		{"c4.large", 4}, {"c4.xlarge", 8}, {"c4.2xlarge", 16}, {"c4.4xlarge", 32}, {"c4.8xlarge", 64},
		{"i2.large", 4}, {"i2.xlarge", 8}, {"i2.2xlarge", 16}, {"i2.4xlarge", 32}, {"i2.8xlarge", 64},
		{"i3.large", 4}, {"i3.xlarge", 8}, {"i3.2xlarge", 16}, {"i3.4xlarge", 32}, {"i3.8xlarge", 64}, {"i3.16xlarge", 128},
	}},
	{Name: "us-east-1", Zones: 6, Specifications: allSpecs},
	{Name: "us-west-2", Zones: 3, Specifications: allSpecs},
	{Name: "us-west-1", Zones: 2, Specifications: []Specification{
		{"m4.large", 4}, {"m4.xlarge", 8}, {"m4.2xlarge", 16}, {"m4.4xlarge", 32}, {"m4.10xlarge", 80}, {"m4.16xlarge", 128},
		{"c4.large", 4}, {"c4.xlarge", 8}, {"c4.2xlarge", 16}, {"c4.4xlarge", 32}, {"c4.8xlarge", 64},
		{"t2.small", 1}, {"t2.medium", 2}, {"t2.large", 4}, {"t2.xlarge", 8},
		{"m3.medium", 2}, {"m3.large", 4}, {"m3.xlarge", 8}, {"m3.2xlarge", 16},
		{"c3.large", 4}, {"c3.xlarge", 8}, {"c3.2xlarge", 16}, {"c3.4xlarge", 32}, {"c3.8xlarge", 64},
		{"i2.large", 4}, {"i2.xlarge", 8}, {"i2.2xlarge", 16}, {"i2.4xlarge", 32}, {"i2.8xlarge", 64},
		{"i3.large", 4}, {"i3.xlarge", 8}, {"i3.2xlarge", 16}, {"i3.4xlarge", 32}, {"i3.8xlarge", 64}, {"i3.16xlarge", 128},
	}},
	{Name: "eu-west-2", Zones: 2, Specifications: []Specification{
		{"t2.small", 1}, {"t2.medium", 2}, {"t2.large", 4}, {"t2.xlarge", 8},
		{"m4.large", 4}, {"m4.xlarge", 8}, {"m4.2xlarge", 16}, {"m4.4xlarge", 32}, {"m4.10xlarge", 80}, {"m4.16xlarge", 128},
		{"c4.large", 4}, {"c4.xlarge", 8}, {"c4.2xlarge", 16}, {"c4.4xlarge", 32}, {"c4.8xlarge", 64},
		{"i3.large", 4}, {"i3.xlarge", 8}, {"i3.2xlarge", 16}, {"i3.4xlarge", 32}, {"i3.8xlarge", 64}, {"i3.16xlarge", 128},
	}},
	{Name: "eu-west-1", Zones: 3, Specifications: allSpecs},
	{Name: "eu-central-1", Zones: 3, Specifications: allSpecs},
	{Name: "ap-northeast-2", Zones: 2, Specifications: allSpecs},
	{Name: "ap-northeast-1", Zones: 2, Specifications: allSpecs},
	{Name: "ap-southeast-2", Zones: 3, Specifications: []Specification{
		{"m4.large", 4}, {"m4.xlarge", 8}, {"m4.2xlarge", 16}, {"m4.4xlarge", 32}, {"m4.10xlarge", 80}, {"m4.16xlarge", 128},
		{"c4.large", 4}, {"c4.xlarge", 8}, {"c4.2xlarge", 16}, {"c4.4xlarge", 32}, {"c4.8xlarge", 64},
		{"t2.small", 1}, {"t2.medium", 2}, {"t2.large", 4}, {"t2.xlarge", 8},
		{"m3.medium", 2}, {"m3.large", 4}, {"m3.xlarge", 8}, {"m3.2xlarge", 16},
		{"c3.large", 4}, {"c3.xlarge", 8}, {"c3.2xlarge", 16}, {"c3.4xlarge", 32}, {"c3.8xlarge", 64},
		{"i2.large", 4}, {"i2.xlarge", 8}, {"i2.2xlarge", 16}, {"i2.4xlarge", 32}, {"i2.8xlarge", 64},
		{"i3.large", 4}, {"i3.xlarge", 8}, {"i3.2xlarge", 16}, {"i3.4xlarge", 32}, {"i3.8xlarge", 64}, {"i3.16xlarge", 128},
	}},
	{Name: "ap-southeast-1", Zones: 2, Specifications: allSpecs},
	{Name: "ca-central-1", Zones: 2, Specifications: allSpecs},
}

type Region struct {
	Name           string
	Zones          int
	Specifications []Specification
}

// Return exactly 50 specs spread across instance size and subnets
func (r Region) SpecificationInstances() []SpecificationInstance {
	var specs []SpecificationInstance

	// The choice here between more types and more availability zones is a tough one!
	// I'm inclined to go with the best instance types (which are first) and then get
	// broad coverage across AZs
	for _, spec := range r.Specifications {
		for i := 0; i < r.Zones; i++ {
			if len(specs) == specsPerTemplate {
				return specs
			}
			if spec.Weight > maxWeight {
				continue
			}
			i := SpecificationInstance{
				Specification: spec,
				SubnetOffset:  i,
			}
			// log.Printf("%d %#v", len(specs), i)
			specs = append(specs, i)
		}
	}

	return specs
}

type Specification struct {
	InstanceType string
	Weight       int
}

type SpecificationInstance struct {
	Specification
	SubnetOffset int
}

func writeRegionTemplate(region Region) error {
	filename := fmt.Sprintf("templates/spotfleet-%s.yaml", region.Name)
	log.Printf("Writing %s", filename)

	b := &bytes.Buffer{}

	t := template.Must(template.New("cloudformation").Parse(yamlTemplate))
	err := t.Execute(b, region.SpecificationInstances())
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, b.Bytes(), 0660)
}

func main() {
	for _, region := range regions {
		if err := writeRegionTemplate(region); err != nil {
			log.Fatal(err)
		}
	}
}
