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

	"github.com/spf13/viper"

  "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

var clusterNetworkTemplate = "cluster-network.json"
var clusterMasterTemplate = "cluster-master.json"
var clusterComputeTemplate = "cluster-compute.json"

var MasterInstanceTypes = []string{
  "small-t2.large",
  "medium-r3.2xlarge",
  "large-c4.8xlarge",
  "gpu-g2.2xlarge",
  "enterprise-x1.32xlarge",
}

func IsValidMasterInstanceType(instanceType string) bool {
  return containsS(MasterInstanceTypes, instanceType)
}

var ComputeInstanceTypes = []string{
  "compute-2C-3.75GB.small-c4.large",
  "compute-8C-15GB.medium-c4.2xlarge",
  "compute-16C-30GB.large-c4.4xlarge",
  "compute-36C-60GB.dedicated-c4.8xlarge",
  "balanced-4C-16GB.small-m4.xlarge",
  "balanced-8C-32GB.medium-m4.2xlarge",
  "balanced-16C-64GB.large-m4.4xlarge",
  "balanced-40C-160GB.dedicated-m4.10xlarge",
  "memory-4C-30GB.small-r3.xlarge",
  "memory-8C-60GB.medium-r3.2xlarge",
  "memory-16C-120GB.large-r3.4xlarge",
  "memory-32C-240GB.dedicated-r3.8xlarge",
  "gpu-1GPU-8C-15GB.small-g2.2xlarge",
  "gpu-4GPU-32C-60GB.medium-g2.8xlarge",
  "gpu-8GPU-32C-488GB.large-p2.8xlarge",
  "gpu-16GPU-64C-732GB.dedicated-p2.16xlarge",
  "enterprise-64C-976GB.large-x1.16xlarge",
  "enterprise-128C-1952GB.dedicated-x1.32xlarge",
}

// network: 20
// master: 12
// compute: 13
var ClusterResourceCount int = 45

func IsValidComputeInstanceType(instanceType string) bool {
  return containsS(ComputeInstanceTypes, instanceType)
}

type Cluster struct {
  Name string
  Domain *Domain
  Network *ClusterNetwork
  Master *Master
  ComputeGroups []*ComputeGroup
  TopicARN string
  MessageHandler func(msg string)
}

type ClusterNetwork struct {
  Index int
  Stack *cloudformation.Stack
}

func (c *ClusterNetwork) NetworkPool() string {
  return getStackParameter(c.Stack, "FlightNetworkingPool")
}

func (c *ClusterNetwork) NetworkIndex() string {
  return getStackParameter(c.Stack, "FlightNetworkingIndex")
}

func (c *ClusterNetwork) PublicSubnet() string {
  return getStackOutput(c.Stack, "FlightPublicSubnet")
}

func (c *ClusterNetwork) ManagementSubnet() string {
  return getStackOutput(c.Stack, "FlightManagementSubnet")
}

func (c *ClusterNetwork) PrivateSubnet() string {
  return getStackOutput(c.Stack, "FlightPrivateSubnet")
}

func (c *ClusterNetwork) PlacementGroup() string {
  return getStackOutput(c.Stack, "FlightPlacementGroup")
}

func (c *ClusterNetwork) PrivateRouteTable() string {
  return getStackOutput(c.Stack, "FlightPrivateRouteTable")
}

type Master struct {
  Stack *cloudformation.Stack
}

func (m *Master) AccessIP() string {
  return getStackOutput(m.Stack, "AccessIP")
}

func (m *Master) PrimaryNetworkInterface() string {
  return getStackOutput(m.Stack, "FlightLoginPrimaryNetworkInterface")
}

func (m *Master) PrivateIP() string {
  return getStackOutput(m.Stack, "FlightLoginPrivateIP")
}

func (m *Master) Username() string {
  return getStackOutput(m.Stack, "Username")
}

func (m *Master) WebAccess() string {
  return getStackOutput(m.Stack, "WebAccess")
}

type ComputeGroup struct {
  Stack *cloudformation.Stack
}

func NewCluster(name string, domain *Domain, handler func(msg string)) *Cluster {
  return &Cluster{name, domain, nil, nil, nil, "", handler}
}

func (c *Cluster) processQueue(qArn *string) {
  for c.MessageHandler != nil {
    time.Sleep(500 * time.Millisecond)
    receiveMessage(qArn, c.MessageHandler)
  }
}

func (c *Cluster) Tags() []*cloudformation.Tag {
  return []*cloudformation.Tag{
    {
      Key: aws.String("flight:cluster"),
      Value: aws.String(c.Name),
    },
  }
}

func (c *Cluster) Create() error {
	svc, err := CloudFormation()
  if err != nil { return err }

  tArn, qUrl, err := setupEventHandling("flight-" + c.Domain.Name + "-cluster-" + c.Name)
  if err != nil { return err }
  go c.processQueue(qUrl)
  c.TopicARN = *tArn

  err = createClusterNetwork(c, svc)
  if err != nil { return err }
  // create master node
  err = createMaster(c, svc)
  if err != nil { return err }
  // create compute group(s)
  err = createComputeGroup(c, svc)
  if err != nil { return err }

  c.MessageHandler("DONE")

  return nil
}

func (c *Cluster) Destroy() error {
	svc, err := CloudFormation()
  if err != nil { return err }

  qUrl, err := getEventQueueUrl("flight-" + c.Domain.Name + "-cluster-" + c.Name)
  if err != nil { return err }
  go c.processQueue(qUrl)

  err = destroyComputeGroup(c, 1, svc)
  if err != nil { return err }
  err = destroyMaster(c, svc)
  if err != nil { return err }
  err = destroyClusterNetwork(c, svc)
  if err != nil { return err }

  err = cleanupEventHandling("flight-" + c.Domain.Name + "-cluster-" + c.Name)
  if err != nil { return err }

  c.MessageHandler("DONE")

  entity, err := c.LoadEntity()
  if err != nil { return err }
  err = c.Domain.ReleaseNetwork(entity.NetworkIndex)
  if err != nil { return err }
  c.DestroyEntity()

  return nil
}

func ClusterNames() ([]string, error) {
  var clusters []string

  err := eachRunningStack(func(stack *cloudformation.Stack) {
    for _, tag := range stack.Tags {
      if *tag.Key == "flight:type" && *tag.Value == "master" {
        clusters = append(clusters, getStackParameter(stack, "ClusterName"))
      }
    }
  })

  return clusters, err
}

func destroyComputeGroup(cluster *Cluster, index int, svc *cloudformation.CloudFormation) error {
  stackName := fmt.Sprintf("flight-%s-%s-compute-%d",
    cluster.Domain.Name,
    cluster.Name,
    index)

  return destroyStack(svc, stackName)
}

func destroyMaster(cluster *Cluster, svc *cloudformation.CloudFormation) error {
  stackName := fmt.Sprintf("flight-%s-%s-master", cluster.Domain.Name, cluster.Name)
  return destroyStack(svc, stackName)
}

func destroyClusterNetwork(cluster *Cluster, svc *cloudformation.CloudFormation) error {
  stackName := fmt.Sprintf("flight-%s-%s-network", cluster.Domain.Name, cluster.Name)
  return destroyStack(svc, stackName)
}

func createMaster(cluster *Cluster, svc *cloudformation.CloudFormation) error {
  launchParams, err := createMasterLaunchParameters(cluster)
  if err != nil { return err }

  stackName := fmt.Sprintf("flight-%s-%s-master", cluster.Domain.Name, cluster.Name)
  url := fmt.Sprintf(TemplateSets[Config().TemplateSet],clusterMasterTemplate)
  stack, err := createStack(svc, launchParams, cluster.Tags(), url, stackName, "master", cluster.TopicARN, cluster.Domain)
  if err != nil { return err }

  cluster.Master = &Master{stack}
  return nil
}

func createComputeGroup(cluster *Cluster, svc *cloudformation.CloudFormation) error {
  launchParams, err := createComputeLaunchParameters(cluster)
  if err != nil { return err }

  stackName := fmt.Sprintf("flight-%s-%s-compute-%d",
    cluster.Domain.Name,
    cluster.Name,
    len(cluster.ComputeGroups) + 1)

  url := fmt.Sprintf(TemplateSets[Config().TemplateSet],clusterComputeTemplate)
  stack, err := createStack(svc, launchParams, cluster.Tags(), url, stackName, "compute", cluster.TopicARN, cluster.Domain)
  if err != nil { return err }

  cluster.ComputeGroups = append(cluster.ComputeGroups, &ComputeGroup{stack})
  return nil
}

func createClusterNetwork(cluster *Cluster, svc *cloudformation.CloudFormation) error {
  network, err := cluster.Domain.BookNetwork()
  if err != nil { return err }

  launchParams, err := createNetworkLaunchParameters(cluster, network)
  if err != nil { return err }

  stackName := fmt.Sprintf("flight-%s-%s-network", cluster.Domain.Name, cluster.Name)

  url := fmt.Sprintf(TemplateSets[Config().TemplateSet],clusterNetworkTemplate)
  tags := append(cluster.Tags(), &cloudformation.Tag{Key: aws.String("flight:network"), Value: aws.String(strconv.Itoa(network))})
  stack, err := createStack(svc, launchParams, tags, url, stackName, "network", cluster.TopicARN, cluster.Domain)
  if err != nil { return err }

  cluster.Network = &ClusterNetwork{network, stack}
  err = cluster.CreateEntity()
  if err != nil { return err }

  return nil
}

func createNetworkLaunchParameters(cluster *Cluster, network int) ([]*cloudformation.Parameter, error) {
  networkPool := (network / 32) + 1
  networkIndex := (network % 32) + 1

  params := []*cloudformation.Parameter{
    {
      ParameterKey: aws.String("FlightVPC"),
      ParameterValue: aws.String(cluster.Domain.VPC()),
    },
    {
      ParameterKey: aws.String("FlightNetworkingPool"),
      ParameterValue: aws.String(strconv.Itoa(networkPool)),
    },
    {
      ParameterKey: aws.String("FlightNetworkingIndex"),
      ParameterValue: aws.String(strconv.Itoa(networkIndex)),
    },
    {
      ParameterKey: aws.String("FlightPublicRouteTable"),
      ParameterValue: aws.String(cluster.Domain.PublicRouteTable()),
    },
  }
  return params, nil
}

func createMasterLaunchParameters(cluster *Cluster) ([]*cloudformation.Parameter, error) {
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
      ParameterKey: aws.String("AccessUsername"),
      ParameterValue: aws.String(viper.GetString("admin-user-name")),
    },
    {
      ParameterKey: aws.String("ClusterName"),
      ParameterValue: aws.String(cluster.Name),
    },
    {
      ParameterKey: aws.String("FlightVPC"),
      ParameterValue: aws.String(cluster.Domain.VPC()),
    },
    {
      ParameterKey: aws.String("FlightDomain"),
      ParameterValue: aws.String(cluster.Domain.Prefix()),
    },
    {
      ParameterKey: aws.String("FlightNetworkingPool"),
      ParameterValue: aws.String(cluster.Network.NetworkPool()),
    },
    {
      ParameterKey: aws.String("FlightNetworkingIndex"),
      ParameterValue: aws.String(cluster.Network.NetworkIndex()),
    },
    {
      ParameterKey: aws.String("FlightPublicSubnet"),
      ParameterValue: aws.String(cluster.Network.PublicSubnet()),
    },
    {
      ParameterKey: aws.String("FlightManagementSubnet"),
      ParameterValue: aws.String(cluster.Network.ManagementSubnet()),
    },
    {
      ParameterKey: aws.String("FlightPlacementGroup"),
      ParameterValue: aws.String(cluster.Network.PlacementGroup()),
    },
    {
      ParameterKey: aws.String("FlightFeatures"),
      ParameterValue: aws.String("password-auth"),
    },
    {
      ParameterKey: aws.String("FlightProfileBucket"),
      ParameterValue: aws.String(""),
    },
    {
      ParameterKey: aws.String("FlightProfiles"),
      ParameterValue: aws.String(""),
    },
  }
  return params, nil
}

func createComputeLaunchParameters(cluster *Cluster) ([]*cloudformation.Parameter, error) {
  params := []*cloudformation.Parameter{
    {
      ParameterKey: aws.String("AccessKeyName"),
      ParameterValue: aws.String(Config().AccessKeyName),
    },
    {
      ParameterKey: aws.String("ClusterName"),
      ParameterValue: aws.String(cluster.Name),
    },
    {
      ParameterKey: aws.String("FlightVPC"),
      ParameterValue: aws.String(cluster.Domain.VPC()),
    },
    {
      ParameterKey: aws.String("FlightLoginPrimaryNetworkInterface"),
      ParameterValue: aws.String(cluster.Master.PrimaryNetworkInterface()),
    },
    {
      ParameterKey: aws.String("FlightDomain"),
      ParameterValue: aws.String(cluster.Domain.Prefix()),
    },
    {
      ParameterKey: aws.String("FlightNetworkingPool"),
      ParameterValue: aws.String(cluster.Network.NetworkPool()),
    },
    {
      ParameterKey: aws.String("FlightNetworkingIndex"),
      ParameterValue: aws.String(cluster.Network.NetworkIndex()),
    },
    {
      ParameterKey: aws.String("ComputeSpotPrice"),
      ParameterValue: aws.String("0.5"),
    },
    {
      ParameterKey: aws.String("ComputeAutoscalingPolicy"),
      ParameterValue: aws.String("enabled"),
    },
    {
      ParameterKey: aws.String("ComputeInitialNodes"),
      ParameterValue: aws.String("1"),
    },
    {
      ParameterKey: aws.String("FlightPrivateSubnet"),
      ParameterValue: aws.String(cluster.Network.PrivateSubnet()),
    },
    {
      ParameterKey: aws.String("FlightManagementSubnet"),
      ParameterValue: aws.String(cluster.Network.ManagementSubnet()),
    },
    {
      ParameterKey: aws.String("FlightPlacementGroup"),
      ParameterValue: aws.String(cluster.Network.PlacementGroup()),
    },
    {
      ParameterKey: aws.String("FlightFeatures"),
      ParameterValue: aws.String(""),
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
      ParameterKey: aws.String("FlightLoginPrivateIP"),
      ParameterValue: aws.String(cluster.Master.PrivateIP()),
    },
    {
      ParameterKey: aws.String("FlightPrivateRouteTable"),
      ParameterValue: aws.String(cluster.Network.PrivateRouteTable()),
    },
  }
  return params, nil
}