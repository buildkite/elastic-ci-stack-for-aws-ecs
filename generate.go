package main

import (
	"bytes"
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

  KeyName:
    Type: String
    Default: ""

  CloudInitScript:
    Description: A URL to a cloud init script to be included in the instance user-data
    Type: String


Conditions:
  HasKeyName: !Not [ !Equals [ !Ref KeyName, "" ] ]

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
            KeyName: !If [HasKeyName, !Ref KeyName, !Ref "AWS::NoValue"]
            SecurityGroups:
              - GroupId: !GetAtt ECSSecurityGroup.GroupId
            SubnetId: !Join [", ", !Ref Subnets]
            UserData:
              "Fn::Base64": !Sub |
                Content-Type: multipart/mixed; boundary="==BOUNDARY=="
                MIME-Version: 1.0

                --==BOUNDARY==
                Content-Type: text/x-shellscript; charset="us-ascii"

                #!/bin/bash
                yum install -y aws-cfn-bootstrap
                echo ECS_CLUSTER=${ECSCluster} >> /etc/ecs/ecs.config
                /opt/aws/bin/cfn-signal -e $? --stack ${AWS::StackName} --resource SpotFleet --region ${AWS::Region}

                --==BOUNDARY==--
                Content-Type: text/x-include-url; charset="us-ascii"
                #!/bin/bash

                [${CloudInitScript}]
                --==BOUNDARY==--
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

// from https://aws.amazon.com/ec2/spot/pricing/

var allSpecs = []Specification{
	{"m5.large", 4}, {"m5.xlarge", 8}, {"m5.2xlarge", 16}, {"m5.4xlarge", 32},
	{"c5.large", 4}, {"c5.xlarge", 8}, {"c5.2xlarge", 16}, {"c5.4xlarge", 32},
	{"t2.small", 1}, {"t2.medium", 2}, {"t2.large", 4}, {"t2.xlarge", 8},
	{"m4.large", 4}, {"m4.xlarge", 8}, {"m4.2xlarge", 16}, {"m4.4xlarge", 32},
	{"c4.large", 4}, {"c4.xlarge", 8}, {"c4.2xlarge", 16}, {"c4.4xlarge", 32},

	// These use instance store, so we'll leave them out for now
	// {"c3.large", 4}, {"c3.xlarge", 8}, {"c3.2xlarge", 16}, {"c3.4xlarge", 32}, {"c3.8xlarge", 64},
	// {"i2.large", 4}, {"i2.xlarge", 8}, {"i2.2xlarge", 16}, {"i2.4xlarge", 32}, {"i2.8xlarge", 64},
	// {"i3.large", 4}, {"i3.xlarge", 8}, {"i3.2xlarge", 16}, {"i3.4xlarge", 32}, {"i3.8xlarge", 64}, {"i3.16xlarge", 128},
}

type Specification struct {
	InstanceType string
	Weight       int
}

func writeTemplate(filename string, specs []Specification) error {
	b := &bytes.Buffer{}

	t := template.Must(template.New("cloudformation").Parse(yamlTemplate))
	err := t.Execute(b, specs)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, b.Bytes(), 0660)
}

func main() {
	if err := writeTemplate("templates/ecs-spotfleet.yaml", allSpecs); err != nil {
		log.Fatal(err)
	}
}
