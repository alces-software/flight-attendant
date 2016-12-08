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
  "strings"
  "encoding/json"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws/credentials"
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

func createStack(
  svc *cloudformation.CloudFormation,
  params []*cloudformation.Parameter,
  tags []*cloudformation.Tag,
  templateUrl string,
  stackName string,
  stackType string,
  topicArn string,
  domain *Domain) (*cloudformation.Stack, error) {

  createParams := &cloudformation.CreateStackInput{
    Capabilities: []*string{aws.String("CAPABILITY_IAM")},
    NotificationARNs: []*string{aws.String(topicArn)},
		StackName: aws.String(stackName),
		TemplateURL: aws.String(templateUrl),
    Parameters: params,
		Tags: append(tags, []*cloudformation.Tag{
      {
				Key: aws.String("flight:domain"),
				Value: aws.String(domain.Name),
			},
      {
        Key: aws.String("flight:type"),
        Value: aws.String(stackType),
      },
    }...),
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
		if strings.HasPrefix(*value.StackName, "flight-") {
			stacksResp, err := svc.DescribeStacks(&cloudformation.DescribeStacksInput{
				StackName: value.StackName,
			})
			if err != nil { return err }
      fn(stacksResp.Stacks[0])
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
      physRes := section.Key("PhysicalResourceId").String()
      if physRes != "" {
        handler(fmt.Sprintf("%s %s (%s)", section.Key("ResourceStatus").String(), section.Key("LogicalResourceId").String(), physRes))
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
