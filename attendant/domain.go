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
  "strconv"
  "time"
  
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

var DomainResourceCount int = 34

type Domain struct {
  Name string
  Stack *cloudformation.Stack
  MessageHandler func(msg string)
}

type DomainStatus struct {
  Clusters map[string]*Cluster
  Appliances map[string]*Appliance
}

func (d *Domain) Prefix() string {
  if d.Stack == nil {
    err := d.AssertExists()
    if err != nil {
      return ""
    }
  }
  var prefix string
  for _, tag := range d.Stack.Tags {
    if *tag.Key == "flight:prefix" {
      prefix = *tag.Value
    }
  }
  return prefix
}

func (d *Domain) VPC() string {
  return d.getOutput("FlightVPC")
}

func (d *Domain) PublicSubnet() string {
  return d.getOutput("FlightPublicSubnet")
}

func (d *Domain) ManagementSubnet() string {
  return d.getOutput("FlightManagementSubnet")
}

func (d *Domain) PublicRouteTable() string {
  return d.getOutput("FlightPublicRouteTable")
}

func (d *Domain) PrivateRouteTable() string {
  return d.getOutput("FlightPrivateRouteTable")
}

func (d *Domain) getOutput(key string) string {
  if d.Stack == nil {
    if err := d.AssertExists(); err != nil {
      return ""
    }
  }
  return getStackOutput(d.Stack, key)
}

func SoloStatus() (*DomainStatus, error) {
  var soloStatus = DomainStatus{make(map[string]*Cluster),make(map[string]*Appliance)}
  err := eachRunningStack(func(stack *cloudformation.Stack) {
    stackType := getStackTag(stack, "flight:type")
    if stackType == "solo" {
      clusterName := getStackTag(stack, "flight:cluster")
      cluster := &Cluster{Name: clusterName, Master: &Master{stack}}
      soloStatus.Clusters[clusterName] = cluster
    }
  })
  return &soloStatus, err
}

func (d *Domain) Status() (*DomainStatus, error) {
  err := d.AssertExists()
  if err != nil { return nil, err }
  var runningInfra = DomainStatus{make(map[string]*Cluster),make(map[string]*Appliance)}
  // check no infrastructure or clusters exist in domain
  err = eachRunningStack(func(stack *cloudformation.Stack) {
    for _, tag := range stack.Tags {
      if *tag.Key == "flight:domain" && *tag.Value == d.Name {
        stackType := getStackTag(stack, "flight:type")
        switch stackType {
        case "master", "network", "compute":
          clusterName := getStackTag(stack, "flight:cluster")
          cluster, exists := runningInfra.Clusters[clusterName]
          if ! exists {
            cluster = &Cluster{Name: clusterName, Domain: d}
            runningInfra.Clusters[clusterName] = cluster
          }
          if stackType == "master" {
            cluster.Master = &Master{stack}
          } else if stackType == "network" {
            idx, _ := strconv.Atoi(getStackTag(stack,"flight:network"))
            cluster.Network = &ClusterNetwork{idx, stack}
          } else if stackType == "compute" {
            cluster.ComputeGroups = append(cluster.ComputeGroups, &ComputeGroup{stack})
          }
        case "appliance":
          applianceName := getStackTag(stack, "flight:appliance")
          runningInfra.Appliances[applianceName] = &Appliance{applianceName, d, stack, nil}
        }
      }
    }
  })
  return &runningInfra, err
}

func (d *Domain) Destroy() error {
	svc, err := CloudFormation()
  if err != nil { return err }

  if err = d.AssertExists(); err != nil {
    return err
  }

  stackName := "flight-" + d.Name
  qUrl, err := getEventQueueUrl(stackName)
  if err != nil { return err }
  go d.processQueue(qUrl)

  err = destroyStack(svc, stackName)
  if err != nil { return err }

  err = cleanupEventHandling(stackName)
  if err != nil { return err }

  d.MessageHandler("DONE")

  err = d.DestroyEntity()
  return err
}

func (d *Domain) AssertReady() error {
  if err := d.AssertExists(); err != nil {
    return err
  }
  if *d.Stack.StackStatus == "CREATE_IN_PROGRESS" {
    return fmt.Errorf("Domain '%s' is not yet ready.", d.Name)
  }
  return nil
}

func (d *Domain) AssertExists() error {
  if d.Stack != nil {
    return nil
  }
	svc, err := CloudFormation()
  if err != nil { return err }

  stacksResp, err := svc.DescribeStacks(&cloudformation.DescribeStacksInput{
    StackName: aws.String("flight-" + d.Name),
  })
  if err != nil {
    if aerr, ok := err.(awserr.Error); ok {
      switch aerr.Code() {
      case "ValidationError":
        return fmt.Errorf("Domain '%s' was not found.", d.Name)
      }
    }
    return err
  }
  stack := stacksResp.Stacks[0]
  for _, tag := range stack.Tags {
    if *tag.Key == "flight:type" && *tag.Value == "domain" {
      d.Stack = stack
    }
  }
  if d.Stack == nil {
    return fmt.Errorf("Domain '%s' was not found.", d.Name)
  }
  return nil
}

func (d *Domain) processQueue(qUrl *string) {
  for d.MessageHandler != nil {
    time.Sleep(500 * time.Millisecond)
    receiveMessage(qUrl, d.MessageHandler)
  }
}

func (d *Domain) Create(prefix string) error {
	svc, err := CloudFormation()
  if err != nil { return err }

  stackName := "flight-" + d.Name
  tArn, qUrl, err := setupEventHandling(stackName)
  if err != nil { return err }
  go d.processQueue(qUrl)

	params := &cloudformation.CreateStackInput{
		StackName: aws.String(stackName),
		TemplateURL: aws.String(TemplateUrl("domain.json")),
    NotificationARNs: []*string{tArn},
		Tags: []*cloudformation.Tag{
			{
				Key: aws.String("flight:domain"),
				Value: aws.String(d.Name),
			},
			{
				Key: aws.String("flight:prefix"),
				Value: aws.String(prefix),
			},
			{
				Key: aws.String("flight:type"),
				Value: aws.String("domain"),
			},
		},
	}

	_, err = svc.CreateStack(params)
  if err != nil {
    cleanupEventHandling(stackName)
    return err
  }
  stack, err := awaitStack(svc, stackName)
  if err != nil {
    cleanupEventHandling(stackName)
    return err
  }
  d.Stack = stack
  err = d.SaveEntity()

  d.MessageHandler("DONE")

  return err
}

func NewDomain(name string, handler func(msg string)) *Domain {
  return &Domain{name, nil, handler}
}

func AllDomains() ([]Domain, error) {
  var domains []Domain

  err := eachRunningStack(func(stack *cloudformation.Stack) {
    for _, tag := range stack.Tags {
      if *tag.Key == "flight:type" && *tag.Value == "domain" {
        domains = append(domains, Domain{getStackTag(stack, "flight:domain"), stack, nil})
      }
    }
  })

  return domains, err
}

func DefaultDomain() (*Domain, error) {
  domains, err := AllDomains()
  if err != nil { return nil, err }
  if len(domains) == 0 {
    return nil, fmt.Errorf("No domains were found.")
  }
  return &domains[0], nil
}
