// Copyright © 2016 Alces Software Ltd <support@alces-software.com>
// This file is part of Flight Attendant.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This software is distributed in the hope that it will be useful, but
// WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
// Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public
// License along with this software.  If not, see
// <http://www.gnu.org/licenses/>.
//
// This package is available under a dual licensing model whereby use of
// the package in projects that are licensed so as to be compatible with
// AGPL Version 3 may use the package under the terms of that
// license. However, if AGPL Version 3.0 terms are incompatible with your
// planned use of this package, alternative license terms are available
// from Alces Software Ltd - please direct inquiries about licensing to
// licensing@alces-software.com.
//
// For more information, please visit <http://www.alces-software.com/>.
//

package attendant

import (
  "fmt"
  "regexp"
  "strings"
  "encoding/json"
  "github.com/aws/aws-sdk-go/aws"
  "github.com/aws/aws-sdk-go/aws/awserr"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/aws/aws-sdk-go/aws/credentials"
  "github.com/aws/aws-sdk-go/service/autoscaling"
  "github.com/aws/aws-sdk-go/service/cloudformation"
  "github.com/aws/aws-sdk-go/service/ec2"
  "github.com/aws/aws-sdk-go/service/sns"
  "github.com/aws/aws-sdk-go/service/sqs"
  "github.com/guregu/dynamo"
  "github.com/go-ini/ini"
)

var awsSession *session.Session

var sqsPolicyTemplate = `
{
  "Version": "2012-10-17",
  "Id": "%s/SQSDefaultPolicy",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "AWS": "*"
      },
      "Action": "SQS:SendMessage",
      "Resource": "%s",
      "Condition": {
        "ArnEquals": {
          "aws:SourceArn": "%s"
        }
      }
    }
  ]
}`

func AwsSession() (*session.Session, error) {
  if awsSession != nil {
    return awsSession, nil
  }

  var opts session.Options
  var config aws.Config
  var creds *credentials.Credentials
  if Config().AwsAccessKey != "" && Config().AwsSecretKey != "" {
    creds = credentials.NewStaticCredentials(Config().AwsAccessKey, Config().AwsSecretKey, "")
  }
  config = aws.Config{Region: &Config().AwsRegion, Credentials: creds}
  opts = session.Options{Config: config}
  awsSession, err := session.NewSessionWithOptions(opts)
  if err != nil { return nil, err }
  return awsSession, nil
}

func CloudFormation() (*cloudformation.CloudFormation, error) {
  sess, err := AwsSession()
  if err != nil { return nil, err }
  return cloudformation.New(sess), nil
}

func Dynamo() (*dynamo.DB, error) {
  sess, err := AwsSession()
  if err != nil { return nil, err }
  return dynamo.New(sess), nil
}

func EC2() (*ec2.EC2, error) {
  sess, err := AwsSession()
  if err != nil { return nil, err }
  return ec2.New(sess), nil
}

func SNS() (*sns.SNS, error) {
  sess, err := AwsSession()
  if err != nil { return nil, err }
  return sns.New(sess), nil
}

func SQS() (*sqs.SQS, error) {
  sess, err := AwsSession()
  if err != nil { return nil, err }
  return sqs.New(sess), nil
}

func AutoScaling() (*autoscaling.AutoScaling, error) {
  sess, err := AwsSession()
  if err != nil { return nil, err }
  return autoscaling.New(sess), nil
}

func IsValidKeyPairName(name string) bool {
  svc, err := EC2()
  if err != nil { return false }
  resp, err := svc.DescribeKeyPairs(&ec2.DescribeKeyPairsInput{
    KeyNames: []*string{&name},
  })
  if err != nil { return false}
  if len(resp.KeyPairs) == 0 { return false }
  return true
}

func CleanFlightEventHandling(stacks []string, dryrun bool, messageHandler func(string)) error {
  snsSvc, err := SNS()
  if err != nil { return err }
  sqsSvc, err := SQS()
  if err != nil { return err }

  topicsResp, err := snsSvc.ListTopics(&sns.ListTopicsInput{})
  if err != nil { return err }
  for _, topic := range topicsResp.Topics {
    topicName := strings.Split(*topic.TopicArn,":")[5]
    if strings.HasPrefix(topicName, "flight-") {
      if ! containsS(stacks, topicName) {
        messageHandler("🗑  Remove topic: " + topicName)

        subResp, err := snsSvc.ListSubscriptionsByTopic(&sns.ListSubscriptionsByTopicInput{TopicArn: topic.TopicArn})
        if err != nil { return err }
        if !dryrun {
          for _, sub := range subResp.Subscriptions {
            snsSvc.Unsubscribe(&sns.UnsubscribeInput{SubscriptionArn: sub.SubscriptionArn})
          }
          snsSvc.DeleteTopic(&sns.DeleteTopicInput{TopicArn: topic.TopicArn})
        }
      } else {
        messageHandler("✅  Retain topic: " + topicName)
      }
    }
  }

  queueResp, err := sqsSvc.ListQueues(&sqs.ListQueuesInput{})
  if err != nil { return err }
  for _, queue := range queueResp.QueueUrls {
    queueName := strings.Split(*queue,"/")[4]
    if strings.HasPrefix(queueName, "flight-") {
      if ! containsS(stacks, queueName) {
        messageHandler("🗑  Remove queue: " + queueName)
        if !dryrun {
          _, err = sqsSvc.DeleteQueue(&sqs.DeleteQueueInput{QueueUrl: queue})
        }
      } else {
        messageHandler("✅  Retain queue: " + queueName)
      }
    }
  }
  return nil
}

func createStack(
  svc *cloudformation.CloudFormation,
  params []*cloudformation.Parameter,
  tags []*cloudformation.Tag,
  templateUrl string,
  stackName string,
  stackType string,
  topicArn string,
  domain *Domain) (*cloudformation.Stack, error) {

  var stackTags []*cloudformation.Tag
  stackTags = append(tags, []*cloudformation.Tag{
    {
      Key: aws.String("flight:type"),
      Value: aws.String(stackType),
    },
    {
      Key: aws.String("flight:template"),
      Value: aws.String(templateUrl),
    },
  }...)
  if domain != nil {
    stackTags = append(stackTags, &cloudformation.Tag{
      Key: aws.String("flight:domain"),
      Value: aws.String(domain.Name),
    })
  }

  createParams := &cloudformation.CreateStackInput{
    Capabilities: []*string{aws.String("CAPABILITY_IAM")},
    NotificationARNs: []*string{aws.String(topicArn)},
    StackName: aws.String(stackName),
    TemplateURL: aws.String(templateUrl),
    Parameters: params,
    Tags: stackTags,
  }

  _, err := svc.CreateStack(createParams)
  if err != nil { return nil, err }

  return awaitStack(svc, stackName)
}

func awaitStack(svc *cloudformation.CloudFormation, stackName string) (*cloudformation.Stack, error) {
  stackParams := &cloudformation.DescribeStacksInput{StackName: aws.String(stackName)}

  err := svc.WaitUntilStackCreateComplete(stackParams)
  if err != nil { return nil, err }

  stacksResp, err := svc.DescribeStacks(stackParams)
  if err != nil { return nil, err }

  return stacksResp.Stacks[0], nil
}

func destroyStack(svc *cloudformation.CloudFormation, stackName string) error {
  deleteParams := &cloudformation.DeleteStackInput{StackName: aws.String(stackName)}
  _, err := svc.DeleteStack(deleteParams)
  if err != nil { return err }

  stackParams := &cloudformation.DescribeStacksInput{StackName: aws.String(stackName)}
  err = svc.WaitUntilStackDeleteComplete(stackParams)
  if err != nil { return err }

  return nil
}

func destroyDetachedNICs(subnetId string) error {
  svc, err := EC2()
  if err != nil { return err }
  // list NICs for subnet
  resp, err := svc.DescribeNetworkInterfaces(&ec2.DescribeNetworkInterfacesInput{
    Filters: []*ec2.Filter{
      &ec2.Filter{
        Name: aws.String("subnet-id"),
        Values: []*string{aws.String(subnetId)},
      },
      &ec2.Filter{
        Name: aws.String("attachment.status"),
        Values: []*string{aws.String("detached")},
      },
    },
  })
  if err != nil { return err }
  for _, nic := range resp.NetworkInterfaces {
    _, err := svc.DeleteNetworkInterface(&ec2.DeleteNetworkInterfaceInput{
      NetworkInterfaceId: nic.NetworkInterfaceId,
    })
    if err != nil { return err }
  }
  return nil
}

func getStack(svc *cloudformation.CloudFormation, stackName string) (*cloudformation.Stack, error) {
  stackParams := &cloudformation.DescribeStacksInput{StackName: aws.String(stackName)}

  stacksResp, err := svc.DescribeStacks(stackParams)
  if err != nil { return nil, err }

  return stacksResp.Stacks[0], nil
}

func getStackParameter(stack *cloudformation.Stack, key string) string {
  var v string
  for _, param := range stack.Parameters {
    if *param.ParameterKey == key {
      v = *param.ParameterValue
      break
    }
  }
  return v
}

func getStackTag(stack *cloudformation.Stack, key string) string {
  var v string
  for _, tag := range stack.Tags {
    if *tag.Key == key {
      v = *tag.Value
      break
    }
  }
  return v
}

func getStackOutput(stack *cloudformation.Stack, key string) string {
  var v string
  for _, output := range stack.Outputs {
    if *output.OutputKey == key {
      v = *output.OutputValue
      break
    }
  }
  return v
}

func getStackConfigValue(stack *cloudformation.Stack, key string) string {
  var v string
  configResult := getStackOutput(stack, "ConfigurationResult")
  if configResult != "" {
    configData := strings.Split(strings.Split(configResult, "\"")[3],";")
    for _, configDatum := range configData {
      configTuple := strings.Split(configDatum, ":")
      if len(configTuple) == 1 {
        configTuple = strings.Split(configDatum, "=")
      }
      if configTuple[0] == key {
        if len(configTuple) == 1 {
          v = "true"
        } else {
          v = strings.TrimSpace(configTuple[1])
        }
        break
      }
    }
  }
  return v
}

func OtherStacks() ([]*cloudformation.Stack, error) {
  var otherStacks = []*cloudformation.Stack{}
  err := eachRunningStackAll(func(stack *cloudformation.Stack) {
    if getStackTag(stack, "flight:type") == "" {
      otherStacks = append(otherStacks, stack)
    }
  })
  return otherStacks, err
}

func getComponentStacksForCluster(cluster *Cluster) ([]*cloudformation.Stack, error) {
  var componentStacks = []*cloudformation.Stack{}
  err := eachRunningStack(func(stack *cloudformation.Stack) {
    if getStackTag(stack, "flight:type") == "component" &&
      getStackTag(stack, "flight:cluster") == cluster.Name &&
      getStackTag(stack, "flight:domain") == cluster.Domain.Name {
      componentStacks = append(componentStacks, stack)
    }
  })
  return componentStacks, err
}

func getComputeGroupStacksForCluster(cluster *Cluster) ([]*cloudformation.Stack, error) {
  var computeGroupStacks = []*cloudformation.Stack{}
  err := eachRunningStack(func(stack *cloudformation.Stack) {
    if getStackTag(stack, "flight:type") == "compute" &&
      getStackTag(stack, "flight:cluster") == cluster.Name &&
      getStackTag(stack, "flight:domain") == cluster.Domain.Name {
      computeGroupStacks = append(computeGroupStacks, stack)
    }
  })
  return computeGroupStacks, err
}

func eachRunningStackAll(fn func(stack *cloudformation.Stack)) error {
  svc, err := CloudFormation()
  if err != nil { return err }

  listParams := &cloudformation.ListStacksInput{
    StackStatusFilter: []*string{
      aws.String("CREATE_COMPLETE"),
      aws.String("CREATE_IN_PROGRESS"),
    },
  }

  resp, err := svc.ListStacks(listParams)
  if err != nil { return err }

  for _, value := range resp.StackSummaries {
    var stacksResp *cloudformation.DescribeStacksOutput
    getter := func() {
      stacksResp, err = svc.DescribeStacks(&cloudformation.DescribeStacksInput{
        StackName: value.StackName,
      })
    }
    getter()
    if err != nil {
      if strings.HasPrefix(err.Error(), "Throttling: Rate exceeded") {
        getter()
      } else if strings.HasPrefix(err.Error(), "AccessDenied") {
        continue
      } else {
        return err
      }
    }
    fn(stacksResp.Stacks[0])
  }

  return err
}

func eachRunningStack(fn func(stack *cloudformation.Stack)) error {
  svc, err := CloudFormation()
  if err != nil { return err }

  listParams := &cloudformation.ListStacksInput{
    StackStatusFilter: []*string{
      aws.String("CREATE_COMPLETE"),
      aws.String("CREATE_IN_PROGRESS"),
    },
  }

  resp, err := svc.ListStacks(listParams)
  if err != nil { return err }

  for _, value := range resp.StackSummaries {
    var stacksResp *cloudformation.DescribeStacksOutput
    if strings.HasPrefix(*value.StackName, "flight-") {
      getter := func() {
        stacksResp, err = svc.DescribeStacks(&cloudformation.DescribeStacksInput{
          StackName: value.StackName,
        })
      }
      getter()
      if err != nil {
        if strings.HasPrefix(err.Error(), "Throttling: Rate exceeded") {
          getter()
        } else if strings.HasPrefix(err.Error(), "AccessDenied") {
          continue
        } else {
          return err
        }
      }
      if len(stacksResp.Stacks) > 0 {
        fn(stacksResp.Stacks[0])
      }
    }
  }

  return err
}

func getEventQueueUrl(name string) (*string, error) {
  sqsSvc, err := SQS()
  if err != nil { return nil, err }
  queueResp, err := sqsSvc.CreateQueue(&sqs.CreateQueueInput{QueueName: &name})
  if err != nil { return nil, err }
  return queueResp.QueueUrl, nil
}

func getEventTopic(name string) (*string, error) {
  snsSvc, err := SNS()
  if err != nil { return nil, err }

  topicResp, err := snsSvc.CreateTopic(&sns.CreateTopicInput{Name: &name})
  if err != nil { return nil, err }
  return topicResp.TopicArn, nil
}

func cleanupEventHandling(stackName string) error {
  snsSvc, err := SNS()
  if err != nil { return err }
  sqsSvc, err := SQS()
  if err != nil { return err }

  tArn, err := getEventTopic(stackName)
  if err != nil { return err }

  qUrl, err := getEventQueueUrl(stackName)
  if err != nil { return err }

  resp, err := snsSvc.ListSubscriptionsByTopic(&sns.ListSubscriptionsByTopicInput{TopicArn: tArn})
  if err != nil { return err }

  for _, sub := range resp.Subscriptions {
    snsSvc.Unsubscribe(&sns.UnsubscribeInput{SubscriptionArn: sub.SubscriptionArn})
  }

  snsSvc.DeleteTopic(&sns.DeleteTopicInput{TopicArn: tArn})
  sqsSvc.DeleteQueue(&sqs.DeleteQueueInput{QueueUrl: qUrl})
  return nil
}

func setupEventHandling(stackName string) (*string, *string, error) {
  snsSvc, err := SNS()
  if err != nil { return nil, nil, err }
  sqsSvc, err := SQS()
  if err != nil { return nil, nil, err }

  var tArn, qUrl *string

  cleanUp := func() {
    // clean up topic
    if tArn != nil {
      snsSvc.DeleteTopic(&sns.DeleteTopicInput{TopicArn: tArn})
    }
    // clean up queue
    if qUrl != nil {
      sqsSvc.DeleteQueue(&sqs.DeleteQueueInput{QueueUrl: qUrl})
    }
  }

  tArn, err = getEventTopic(stackName)
  if err != nil { return nil, nil, err }
  qUrl, err = getEventQueueUrl(stackName)
  if err != nil { cleanUp(); return nil, nil, err }

  attrsResp, err := sqsSvc.GetQueueAttributes(&sqs.GetQueueAttributesInput{
    AttributeNames: []*string{aws.String("QueueArn")},
    QueueUrl: qUrl,
  })
  if err != nil { cleanUp(); return nil, nil, err }
  qArn := attrsResp.Attributes["QueueArn"]
  
  _, err = sqsSvc.SetQueueAttributes(&sqs.SetQueueAttributesInput{
    Attributes: map[string]*string{"Policy": aws.String(fmt.Sprintf(sqsPolicyTemplate, *qArn, *qArn, *tArn))},
    QueueUrl: qUrl,
  })
  if err != nil { cleanUp(); return nil, nil, err }
  
  _, err = snsSvc.Subscribe(&sns.SubscribeInput{Endpoint: qArn, Protocol: aws.String("sqs"), TopicArn: tArn})
  if err != nil { cleanUp(); return nil, nil, err }

  return tArn, qUrl, nil
}

func receiveMessage(qUrl *string, handler func(msg string)) {
  svc, err := SQS()
  if err != nil {
    fmt.Println("Error: " + err.Error())
    return
  }
  resp, err := svc.ReceiveMessage(&sqs.ReceiveMessageInput{QueueUrl: qUrl, MaxNumberOfMessages: aws.Int64(10)})
  if err != nil {
    if aerr, ok := err.(awserr.Error); ok {
      switch aerr.Code() {
      case "AWS.SimpleQueueService.NonExistentQueue":
        // this happens when we destroy the queue while processing messages
        return
      default:
        fmt.Println("Error: " + err.Error())
        return
      }
    }
  }
  for _, message := range resp.Messages {
    var data NotificationMessage
    err = json.Unmarshal([]byte(*message.Body), &data)
    if err != nil {
      fmt.Println("Error: " + err.Error())
      return
    }
    if data.Subject == "AWS CloudFormation Notification" {
      cfg, err := ini.Load([]byte(strings.Replace(data.Message, "\n'", "'", -1)))
      if err != nil {
        fmt.Println("Error: " + err.Error())
        return
      }
      section := cfg.Section("")
      var physResStr, logicalResStr string
      physRes := section.Key("PhysicalResourceId")
      if physRes != nil {
        physResStr = physRes.String()
        if physResStr != "" {
          logicalRes := section.Key("LogicalResourceId")
          if logicalRes != nil {
            logicalResStr = logicalRes.String()
            resStatus := section.Key("ResourceStatus")
            if resStatus != nil {
              handler(fmt.Sprintf("%s %s (%s)", resStatus.String(), logicalResStr, physResStr))
            }
          }
        }
      }
    }
    _, err = svc.DeleteMessage(&sqs.DeleteMessageInput{QueueUrl: qUrl, ReceiptHandle: message.ReceiptHandle})
    if err != nil {
      fmt.Println("Error: " + err.Error())
    }
  }
}

type NotificationMessage struct {
  Type string
  MessageId string
  TopicArn string
  Subject string
  Message string
  Timestamp string
  SignatureVersion string
  Signature string
  SigningCertURL string
  UnsubscribeURL string
}

func describeVPNConnection(connectionId string) (*string, error) {
  svc, err := EC2()
  if err != nil { return nil, err }
  // list NICs for subnet
  resp, err := svc.DescribeVpnConnections(&ec2.DescribeVpnConnectionsInput{
    VpnConnectionIds: []*string{aws.String(connectionId)},
  })
  if err != nil { return nil, err }
  return resp.VpnConnections[0].CustomerGatewayConfiguration, nil
}

func describeAutoscalingGroup(name string) (*autoscaling.Group, error) {
  svc, err := AutoScaling()
  if err != nil { return nil, err }
  resp, err := svc.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{
    AutoScalingGroupNames: []*string{aws.String(name)},
  })
  if err != nil { return nil, err }
  return resp.AutoScalingGroups[0], nil
}

func getStackResources(stack *cloudformation.Stack) ([]*cloudformation.StackResourceSummary, error) {
  svc, err := CloudFormation()
  if err != nil { return nil, err }
  resp, err := svc.ListStackResources(&cloudformation.ListStackResourcesInput{
    StackName: stack.StackName,
  })
  if err != nil { return nil, err }
  return resp.StackResourceSummaries, nil
}

func getAutoscalingResource(stack *cloudformation.Stack) (*cloudformation.StackResourceSummary, error) {
  resources, err := getStackResources(stack)
  if err != nil { return nil, err }
  for _, res := range resources {
    if *res.ResourceType == "AWS::AutoScaling::AutoScalingGroup" {
      return res, nil
    }
  }
  return nil, fmt.Errorf("Autoscaling resource not found for stack: %s", *stack.StackName)
}

func PreflightCheck() error {
  matched, err := regexp.Match("^[a-z]{2}-[a-z]+-[1-9]$",[]byte(Config().AwsRegion))
  if err != nil { return err }
  if !matched {
    return fmt.Errorf("Bad region: %s", Config().AwsRegion)
  }
  svc, err := CloudFormation()
  if err != nil { return err }
  _, err = svc.DescribeAccountLimits(&cloudformation.DescribeAccountLimitsInput{})
  if err != nil {
    if aerr, ok := err.(awserr.Error); ok {
      switch aerr.Code() {
      case "InvalidClientTokenId":
        // this happens when credentials are incorrect
        return fmt.Errorf("Unable to connect to AWS: invalid credentials")
      default:
        return fmt.Errorf("Unable to connect to AWS: connection to endpoint failed")
      }
    }
  }
  return nil
}
