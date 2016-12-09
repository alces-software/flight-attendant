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
  "time"
  
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

type Appliance struct {
  Name string
  Domain *Domain
  Stack *cloudformation.Stack
  MessageHandler func(msg string)
}

var ApplianceTemplates = map[string]string{
  "directory": "directory.json",
  "storage-manager": "storage-manager.json",
  "access-manager": "access-manager.json",
}

var ApplianceResourceCounts = map[string]int {
  "directory": 11,
  "storage-manager": 9,
  "access-manager": 9,
}

var ApplianceInstanceTypes = []string{
  "small-t2.large",
  "medium-r3.large",
  "large-c4.8xlarge",
}

func NewAppliance(name string, domain *Domain, handler func(msg string)) *Appliance {
  return &Appliance{name, domain, nil, handler}
}

func IsValidApplianceInstanceType(instanceType string) bool {
  return containsS(ApplianceInstanceTypes, instanceType)
}

func IsValidApplianceType(applianceType string) bool {
  _, exists := ApplianceTemplates[applianceType]
  return exists
}

func (a *Appliance) Create() error {
	svc, err := CloudFormation()
  if err != nil { return err }

  url := fmt.Sprintf(TemplateSets[Config().TemplateSet],ApplianceTemplates[a.Name])
  if url == "" {
    return fmt.Errorf("Unknown appliance type: %s", a.Name)
  }

  if err = a.Domain.AssertReady(); err != nil { return err }

  var launchParams []*cloudformation.Parameter
  switch a.Name {
  case "directory":
    launchParams = createDirectoryLaunchParameters(a.Domain)
  case "access-manager", "storage-manager":
    launchParams = createApplianceLaunchParameters(a.Domain)
  default:
    return fmt.Errorf("Appliance unsupported: %s", a.Name)
  }

  stackName := fmt.Sprintf("flight-%s-%s", a.Domain.Name, a.Name)
  tArn, qUrl, err := setupEventHandling(stackName)
  if err != nil { return err }
  go a.processQueue(qUrl)
  tags := []*cloudformation.Tag{
    &cloudformation.Tag{Key: aws.String("flight:appliance"), Value: aws.String(a.Name)},
  }
  stack, err := createStack(svc, launchParams, tags, url, stackName, "appliance", *tArn, a.Domain)
  if err != nil { cleanupEventHandling(stackName) }

  a.MessageHandler("DONE")

  a.Stack = stack

  return err
}

func (a Appliance) processQueue(qArn *string) {
  for a.MessageHandler != nil {
    time.Sleep(500 * time.Millisecond)
    receiveMessage(qArn, a.MessageHandler)
  }
}

func (a Appliance) Destroy() error {
	svc, err := CloudFormation()
  if err != nil { return err }

  stackName := fmt.Sprintf("flight-%s-%s", a.Domain.Name, a.Name)
  qUrl, err := getEventQueueUrl(stackName)
  if err != nil { return err }
  go a.processQueue(qUrl)

  err = destroyStack(svc, stackName)
  if err != nil { return err }

  err = cleanupEventHandling(stackName)
  if err != nil { return err }

  a.MessageHandler("DONE")

  return err
}

func createDirectoryLaunchParameters(domain *Domain) []*cloudformation.Parameter {
  params := []*cloudformation.Parameter{
    {
      ParameterKey: aws.String("AccessKeyName"),
      ParameterValue: aws.String(Config().AccessKeyName),
    },
    {
      ParameterKey: aws.String("AccessNetwork"),
      ParameterValue: aws.String("0.0.0.0/0"),
    },
    {
      ParameterKey: aws.String("FlightProfileBucket"),
      ParameterValue: aws.String(""),
    },
    {
      ParameterKey: aws.String("FlightProfiles"),
      ParameterValue: aws.String(""),
    },
    {
      ParameterKey: aws.String("ApplianceInstanceType"),
      ParameterValue: aws.String(Config().ApplianceInstanceType),
    },
    {
      ParameterKey: aws.String("FlightDomain"),
      ParameterValue: aws.String(domain.Prefix()),
    },
    {
      ParameterKey: aws.String("FlightVPC"),
      ParameterValue: aws.String(domain.VPC()),
    },
    {
      ParameterKey: aws.String("FlightPublicSubnet"),
      ParameterValue: aws.String(domain.PublicSubnet()),
    },
    {
      ParameterKey: aws.String("FlightManagementSubnet"),
      ParameterValue: aws.String(domain.ManagementSubnet()),
    },
  }
  return params
}

func createApplianceLaunchParameters(domain *Domain) []*cloudformation.Parameter {
  params := []*cloudformation.Parameter{
    {
      ParameterKey: aws.String("AccessKeyName"),
      ParameterValue: aws.String(Config().AccessKeyName),
    },
    {
      ParameterKey: aws.String("AccessNetwork"),
      ParameterValue: aws.String("0.0.0.0/0"),
    },
    {
      ParameterKey: aws.String("FlightProfileBucket"),
      ParameterValue: aws.String(""),
    },
    {
      ParameterKey: aws.String("FlightProfiles"),
      ParameterValue: aws.String(""),
    },
    {
      ParameterKey: aws.String("ApplianceInstanceType"),
      ParameterValue: aws.String(Config().ApplianceInstanceType),
    },
    {
      ParameterKey: aws.String("FlightDomain"),
      ParameterValue: aws.String(domain.Prefix()),
    },
    {
      ParameterKey: aws.String("FlightVPC"),
      ParameterValue: aws.String(domain.VPC()),
    },
    {
      ParameterKey: aws.String("FlightPublicSubnet"),
      ParameterValue: aws.String(domain.PublicSubnet()),
    },
    {
      ParameterKey: aws.String("FlightFeatures"),
      ParameterValue: aws.String(""),
    },
  }
  return params
}