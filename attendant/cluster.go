// Copyright © 2016-2017 Alces Software Ltd <support@alces-software.com>
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
  "io/ioutil"
  "strconv"
  "strings"
  "time"

  "github.com/spf13/viper"

  "github.com/aws/aws-sdk-go/aws"
  "github.com/aws/aws-sdk-go/aws/awserr"
  "github.com/aws/aws-sdk-go/service/autoscaling"
  "github.com/aws/aws-sdk-go/service/cloudformation"

  "gopkg.in/yaml.v2"
)

var clusterNetworkTemplate = "cluster-network.json"
var clusterMasterTemplate = "cluster-master.json"
var clusterComputeTemplate = "cluster-compute.json"
var soloClusterTemplate = "solo-cluster.json"

var MasterInstanceTypes = []string{
  "small-t2.large",
  "medium-r4.2xlarge",
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
  "memory-4C-30GB.small-r4.xlarge",
  "memory-8C-60GB.medium-r4.2xlarge",
  "memory-16C-120GB.large-r4.4xlarge",
  "memory-32C-240GB.xlarge-r4.8xlarge",
  "memory-64C-480GB.dedicated-r4.16xlarge",
  "gpu-1GPU-8C-15GB.small-g2.2xlarge",
  "gpu-4GPU-32C-60GB.medium-g2.8xlarge",
  "gpu-8GPU-32C-488GB.large-p2.8xlarge",
  "gpu-16GPU-64C-732GB.dedicated-p2.16xlarge",
  "enterprise-64C-976GB.large-x1.16xlarge",
  "enterprise-128C-1952GB.dedicated-x1.32xlarge",
}

// network: 19 (18)
// master: 16 (15)
// compute: 10 (9)
var ClusterResourceCount int = 35
var SoloClusterResourceCount int = 46
var ComputeGroupResourceCount int = 10

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
  ExpiryTime int64
}

type ClusterDetails struct {
  Ip string
  KeyPair string
  Url string
  Username string
  Uuid string
  Token string
  Queues []QueueDetails
  Components []string
  ExpiryTime int64
}

type QueueDetails struct {
  Name string
  InstanceType string
  Pricing string
  ResourceName string
  MaxSize int
  MinSize int
  DesiredCapacity int
  Running int
  ExpiryTime int64
}

type ClusterNetwork struct {
  Index int
  Stack *cloudformation.Stack
}

func (c *ClusterNetwork) NetworkPool() string {
  if c.Stack == nil {
    return strconv.Itoa((c.Index / 32) + 1)
  } else {
    return getStackParameter(c.Stack, "NetworkingPool")
  }
}

func (c *ClusterNetwork) NetworkIndex() string {
  if c.Stack == nil {
    return strconv.Itoa((c.Index % 32) + 1)
  } else {
    return getStackParameter(c.Stack, "NetworkingIndex")
  }
}

func (c *ClusterNetwork) PublicSubnet() string {
  return getStackOutput(c.Stack, "PubSubnet")
}

func (c *ClusterNetwork) ManagementSubnet() string {
  return getStackOutput(c.Stack, "MgtSubnet")
}

func (c *ClusterNetwork) PrivateSubnet() string {
  return getStackOutput(c.Stack, "PrvSubnet")
}

func (c *ClusterNetwork) PlacementGroup() string {
  return getStackOutput(c.Stack, "PlacementGroup")
}

type Master struct {
  Stack *cloudformation.Stack
}

func (m *Master) AccessIP() string {
  return getStackOutput(m.Stack, "AccessIP")
}

func (m *Master) PrivateIP() string {
  return getStackOutput(m.Stack, "MasterPrivateIP")
}

func (m *Master) Username() string {
  return getStackOutput(m.Stack, "Username")
}

func (m *Master) WebAccess() string {
  return getStackOutput(m.Stack, "WebAccess")
}

func (m *Master) ClusterUUID() string {
  return getStackConfigValue(m.Stack, "UUID")
}

func (m *Master) ClusterSecurityToken() string {
  return getStackConfigValue(m.Stack, "Token")
}

type ComputeGroup struct {
  Stack *cloudformation.Stack
  Name string
  InstanceType string
  Pricing string
  ResourceName string
  _AutoscalingGroup *autoscaling.Group
  ExpiryTime int64
}

func NewCluster(name string, domain *Domain, handler func(msg string)) *Cluster {
  return &Cluster{name, domain, nil, nil, nil, "", handler, -1}
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

func (c *Cluster) Create(withQ bool) error {
  svc, err := CloudFormation()
  if err != nil { return err }

  if c.Domain == nil {
    // launch a solo cluster
    tArn, qUrl, err := setupEventHandling("flight-cluster-" + c.Name)
    if err != nil { return err }
    go c.processQueue(qUrl)
    c.TopicARN = *tArn
    err = createSoloCluster(c, svc)
    if err != nil { return err }
  } else {
    tArn, qUrl, err := setupEventHandling("flight-" + c.Domain.Name + "-cluster-" + c.Name)
    if err != nil { return err }
    go c.processQueue(qUrl)
    c.TopicARN = *tArn

    err = createClusterNetwork(c, svc)
    if err != nil { return err }
    // create master node
    err = createMaster(c, svc)
    if err != nil { return err }
  }

  c.MessageHandler("DONE")

  if withQ {
    err = c.AddQueue("default", "", 0)
    if err != nil { return err }
  }
  return nil
}

func (c *Cluster) AddQueue(queueName, queueParamsFile string, expiryTime int64) error {
  svc, err := CloudFormation()
  if err != nil { return err }

  // Load some cluster information, specifically Network
  networkStack, err := getStack(svc, "flight-" + c.Domain.Name + "-" + c.Name + "-network")
  if err != nil { return err }
  idx, err := strconv.Atoi(getStackTag(networkStack, "flight:network"))
  if err != nil { return err }
  c.Network = &ClusterNetwork{idx, networkStack}
  masterStack, err := getStack(svc, "flight-" + c.Domain.Name + "-" + c.Name + "-master")
  if err != nil { return err }
  c.Master = &Master{masterStack}

  tArn, qUrl, err := setupEventHandling("flight-" + c.Domain.Name + "-cluster-" + c.Name)
  if err != nil { return err }
  go c.processQueue(qUrl)
  c.TopicARN = *tArn

  // create compute group(s)
  err = createComputeGroup(c, queueName, queueParamsFile, expiryTime, svc)
  if err != nil { return err }
  c.MessageHandler("DONE")
  return nil
}

func (c *Cluster) DestroyQueue(queueName string) error {
  svc, err := CloudFormation()
  if err != nil { return err }
  qUrl, err := getEventQueueUrl("flight-" + c.Domain.Name + "-cluster-" + c.Name)
  if err != nil { return err }
  go c.processQueue(qUrl)

  err = destroyComputeGroup(c, queueName, svc)
  if err != nil { return err }
  c.MessageHandler("DONE")
  return nil
}

func (c *Cluster) Expand(componentType, componentName, componentParamsFile string) error {
  svc, err := CloudFormation()
  if err != nil { return err }

  // Load some cluster information, specifically Network
  networkStack, err := getStack(svc, "flight-" + c.Domain.Name + "-" + c.Name + "-network")
  if err != nil { return err }
  idx, err := strconv.Atoi(getStackTag(networkStack, "flight:network"))
  if err != nil { return err }
  c.Network = &ClusterNetwork{idx, networkStack}
  masterStack, err := getStack(svc, "flight-" + c.Domain.Name + "-" + c.Name + "-master")
  if err != nil { return err }
  c.Master = &Master{masterStack}
  
  tArn, qUrl, err := setupEventHandling("flight-" + c.Domain.Name + "-cluster-" + c.Name)
  if err != nil { return err }
  go c.processQueue(qUrl)
  c.TopicARN = *tArn

  err = createComponent(componentType, componentName, componentParamsFile, c, svc)
  if err != nil { return err }

  c.MessageHandler("DONE")
  return nil
}

func (c *Cluster) Reduce(componentType, componentName string) error {
  svc, err := CloudFormation()
  if err != nil { return err }

  tArn, qUrl, err := setupEventHandling("flight-" + c.Domain.Name + "-cluster-" + c.Name)
  if err != nil { return err }
  go c.processQueue(qUrl)
  c.TopicARN = *tArn

  err = destroyComponent(c, componentType, componentName, svc)
  if err != nil { return err }

  c.MessageHandler("DONE")
  return nil
}

func (c *Cluster) Purge() error {
  svc, err := CloudFormation()

  if err != nil { return err }
  var ch chan string = make(chan string)
  // purge components
  componentStacks, err := getComponentStacksForCluster(c)
  if err != nil { return err }
  for _, stack := range componentStacks {
    go func(stack *cloudformation.Stack, ch chan<- string) {
      n := fmt.Sprintf("%s %s", *stack.StackName, *stack.StackName)
      c.MessageHandler("DELETE_IN_PROGRESS " + n)
      destroyStack(svc, *stack.StackName)
      ch <- *stack.StackName
    }(stack, ch)
  }

  // purge compute groups
  c.LoadComputeGroups()
  for _, group := range c.ComputeGroups {
    go func(stack *cloudformation.Stack, ch chan<- string) {
      n := fmt.Sprintf("%s %s", *stack.StackName, *stack.StackName)
      c.MessageHandler("DELETE_IN_PROGRESS " + n)
      destroyStack(svc, *stack.StackName)
      ch <- *stack.StackName
    }(group.Stack, ch)
  }

  // purge master
  go func(ch chan<- string) {
    n := fmt.Sprintf("%s %s", *c.Master.Stack.StackName, *c.Master.Stack.StackName)
    c.MessageHandler("DELETE_IN_PROGRESS " + n)
    destroyMaster(c, svc)
    ch <- *c.Master.Stack.StackName
  }(ch)

  count := len(componentStacks) + len(c.ComputeGroups) + 1
  for item := <- ch; item != ""; item = <- ch {
    c.MessageHandler("DELETE_COMPLETE " + item + " " + item)
    count -= 1
    if count == 0 {
      break
    }
  }

  // have to wait until everything else is destroyed before destroing the network
  n := fmt.Sprintf("%s %s", *c.Network.Stack.StackName, *c.Network.Stack.StackName)
  c.MessageHandler("DELETE_IN_PROGRESS " + n)
  destroyClusterNetwork(c, svc)
  c.MessageHandler("DELETE_COMPLETE " + n)

  err = cleanupEventHandling("flight-" + c.Domain.Name + "-cluster-" + c.Name)
  if err != nil { return err }

  entity, err := c.LoadEntity()
  if err != nil { return err }
  err = c.Domain.ReleaseNetwork(entity.NetworkIndex)
  if err != nil { return err }
  c.DestroyEntity()

  return nil
}

func (c *Cluster) Destroy() error {
  svc, err := CloudFormation()
  if err != nil { return err }
  if c.Domain == nil {
    // destroying a solo cluster
    qUrl, err := getEventQueueUrl("flight-cluster-" + c.Name)
    if err != nil { return err }
    go c.processQueue(qUrl)
    err = destroySoloCluster(c, svc)
    if err != nil { return err }

    err = cleanupEventHandling("flight-cluster-" + c.Name)
    if err != nil { return err }

    c.MessageHandler("DONE")
  } else {
    qUrl, err := getEventQueueUrl("flight-" + c.Domain.Name + "-cluster-" + c.Name)
    if err != nil { return err }
    go c.processQueue(qUrl)

    // get any components and destroy them first
    componentStacks, err := getComponentStacksForCluster(c)
    if err != nil { return err }
    c.MessageHandler("DISABLE-COUNTERS")
    for _, stack := range componentStacks {
      err = destroyStack(svc, *stack.StackName)
      if err != nil { return err }
    }
    c.MessageHandler("ENABLE-COUNTERS")

    // get compute group stacks and destroy them next
    c.LoadComputeGroups()
    if err != nil { return err }
    c.MessageHandler(fmt.Sprintf("COUNTERS=%d",ClusterResourceCount + (ComputeGroupResourceCount * len(c.ComputeGroups))))
    for _, group := range c.ComputeGroups {
      err = destroyStack(svc, *group.Stack.StackName)
      if err != nil { return err }
    }

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
  }

  return nil
}

func (c *Cluster) Exists() bool {
  if c.Master == nil {
    if svc, err := CloudFormation(); err == nil {
      if masterStack, err := getStack(svc, "flight-" + c.Domain.Name + "-" + c.Name + "-master"); err == nil {
        c.Master = &Master{masterStack}
      }
    }
  }
  return c.Master != nil
}

func (c *Cluster) LoadComputeGroups() error {
  if len(c.ComputeGroups) > 0 {
    return nil
  }
  stacks, err := getComputeGroupStacksForCluster(c)
  if err != nil { return err }
  for _, stack := range stacks {
    c.ComputeGroups = append(c.ComputeGroups, computeGroupFromStack(stack))
  }
  return nil
}

func computeGroupFromStack(stack *cloudformation.Stack) *ComputeGroup {
  // split after first `compute-`
  var queueName, pricing, resourceName string
  queueNameParts := strings.SplitAfterN(*stack.StackName, "-compute-", 2)
  if len(queueNameParts) > 1 {
    queueName = queueNameParts[1]
  } else {
    queueName = queueNameParts[0]
  }
  instanceType := getStackParameter(stack, "ComputeInstanceType")
  if instanceType == "other" {
    instanceType = getStackParameter(stack, "ComputeInstanceTypeOther")
  }
  spotPrice := getStackParameter(stack, "ComputeSpotPrice")
  if spotPrice != "0" {
    pricing = "<= $" + spotPrice + "/h"
  } else {
    pricing = "on-demand"
  }
  autoscalingResource, _ := getAutoscalingResource(stack)
  if autoscalingResource != nil {
    resourceName = *autoscalingResource.PhysicalResourceId
  }
  expiryTime, _ := strconv.ParseInt(getStackTag(stack, "flight:expiry"), 10, 64)
  return &ComputeGroup{stack,queueName,instanceType,pricing,resourceName,nil,expiryTime}
}

func (c *Cluster) Details() *ClusterDetails {
  var details ClusterDetails = ClusterDetails{}
  if c.Master == nil {
    if svc, err := CloudFormation(); err == nil {
      if masterStack, err := getStack(svc, "flight-" + c.Domain.Name + "-" + c.Name + "-master"); err == nil {
        c.Master = &Master{masterStack}
      }
    }
  }
  if c.Master != nil {
    details.Ip = getStackOutput(c.Master.Stack, "AccessIP")
    if details.Ip == "" {
      details.Ip = getStackOutput(c.Master.Stack, "MasterPrivateIP")
    }
    details.KeyPair = getStackParameter(c.Master.Stack, "AccessKeyName")
    details.Username = getStackOutput(c.Master.Stack, "Username")
    details.Url = getStackOutput(c.Master.Stack, "WebAccess")
    if details.Url == "" {
      details.Url = getStackOutput(c.Master.Stack, "PrivateWebAccess")
    }
    c.LoadComputeGroups()
    componentStacks, _ := getComponentStacksForCluster(c)
    details.Uuid = getStackConfigValue(c.Master.Stack, "UUID")
    if details.Uuid == "" { details.Uuid = "<unknown>" }
    details.Token = getStackConfigValue(c.Master.Stack, "Token")
    if details.Token == "" { details.Token = "<unknown>" }
    details.ExpiryTime, _ = strconv.ParseInt(getStackTag(c.Master.Stack, "flight:expiry"), 10, 64)

    if (len(c.ComputeGroups) > 0) {
      details.Queues = []QueueDetails{}
      for _, group := range c.ComputeGroups {
        details.Queues = append(details.Queues, group.Details())
      }
    }
    if (len(componentStacks) > 0) {
      for _, stack := range componentStacks {
        details.Components = append(details.Components, *stack.StackName)
      }
    }
  }
  return &details
}


func (c *Cluster) GetDetails() string {
  if c.Master == nil {
    if svc, err := CloudFormation(); err == nil {
      if masterStack, err := getStack(svc, "flight-" + c.Domain.Name + "-" + c.Name + "-master"); err == nil {
        c.Master = &Master{masterStack}
      }
    }
  }
  if c.Master != nil {
    ip := getStackOutput(c.Master.Stack, "AccessIP")
    if ip == "" {
      ip = getStackOutput(c.Master.Stack, "MasterPrivateIP")
    }
    keypair := getStackParameter(c.Master.Stack, "AccessKeyName")
    username := getStackOutput(c.Master.Stack, "Username")
    url := getStackOutput(c.Master.Stack, "WebAccess")
    if url == "" {
      url = getStackOutput(c.Master.Stack, "PrivateWebAccess")
    }
    c.LoadComputeGroups()
    componentStacks, _ := getComponentStacksForCluster(c)
    uuid := getStackConfigValue(c.Master.Stack, "UUID")
    if uuid == "" { uuid = "<unknown>" }
    token := getStackConfigValue(c.Master.Stack, "Token")
    if token == "" { token = "<unknown>" }
    details := fmt.Sprintf("Administrator username: %s\nIP address: %s\nKey pair: %s", username, ip, keypair)
    if url != "" {
      details += "\nAccess URL: " + url
    }
    details += fmt.Sprintf("\nUUID: %s\nToken: %s\n", uuid, token)
    expiryTime := c.GetExpiryTime()
    if expiryTime > 0 {
      details += fmt.Sprintf("Expiry: %s\n", time.Unix(expiryTime, 0).Format(time.RFC3339))
    }
    if (len(c.ComputeGroups) > 0) {
      details += "\nQueues: "
      queueDetails := []string{}
      for _, group := range c.ComputeGroups {
        var queueDetail string
        if group.ExpiryTime > 0 {
          queueDetail = fmt.Sprintf("(%s, %s, expiry: %s)", group.InstanceType, group.Pricing, time.Unix(group.ExpiryTime, 0).Format(time.RFC3339))
        } else {
          queueDetail = fmt.Sprintf("(%s, %s)", group.InstanceType, group.Pricing)
        }
        queueDetails = append(queueDetails, group.Name + " " + queueDetail)
      }
      details += strings.Join(queueDetails, ", ") + "\n"
    }
    if (len(componentStacks) > 0) {
      details += "\nComponents: "
      stackNames := []string{}
      for _, stack := range componentStacks {
        stackNames = append(stackNames, *stack.StackName)
      }
      details += strings.Join(stackNames, ", ") + "\n"
    }
    return details
  } else {
    return "(Incomplete)"
  }
}

func (c *Cluster) GetExpiryTime() int64 {
  if c.ExpiryTime == -1 {
    if c.Master == nil {
      if svc, err := CloudFormation(); err == nil {
        if masterStack, err := getStack(svc, "flight-" + c.Domain.Name + "-" + c.Name + "-master"); err == nil {
          c.Master = &Master{masterStack}
        }
      }
    }
    if c.Master != nil {
      c.ExpiryTime, _ = strconv.ParseInt(getStackTag(c.Master.Stack, "flight:expiry"), 10, 64)
    }
  }
  return c.ExpiryTime
}

func (g *ComputeGroup) loadAutoscalingGroup() {
  g._AutoscalingGroup, _ = describeAutoscalingGroup(g.ResourceName)
}

func (g *ComputeGroup) Details() QueueDetails {
  details := QueueDetails{}
  details.Name = g.Name
  details.InstanceType = g.InstanceType
  details.Pricing = g.Pricing
  details.ResourceName = g.ResourceName
  details.ExpiryTime = g.ExpiryTime
  details.MaxSize = g.MaxSize()
  details.MinSize = g.MinSize()
  details.DesiredCapacity = g.DesiredCapacity()
  details.Running = g.Running()
  return details
}

func (g *ComputeGroup) MaxSize() int {
  if g._AutoscalingGroup == nil { g.loadAutoscalingGroup() }
  if g._AutoscalingGroup == nil { return 0 }
  return int(*g._AutoscalingGroup.MaxSize)
}

func (g *ComputeGroup) MinSize() int {
  if g._AutoscalingGroup == nil { g.loadAutoscalingGroup() }
  if g._AutoscalingGroup == nil { return 0 }
  return int(*g._AutoscalingGroup.MinSize)
}

func (g *ComputeGroup) DesiredCapacity() int {
  if g._AutoscalingGroup == nil { g.loadAutoscalingGroup() }
  if g._AutoscalingGroup == nil { return 0 }
  return int(*g._AutoscalingGroup.DesiredCapacity)
}

func (g *ComputeGroup) Running() int {
  if g._AutoscalingGroup == nil { g.loadAutoscalingGroup() }
  if g._AutoscalingGroup == nil { return 0 }
  return len(g._AutoscalingGroup.Instances)
}

func destroyComputeGroup(cluster *Cluster, queueName string, svc *cloudformation.CloudFormation) error {
  stackName := fmt.Sprintf("flight-%s-%s-compute-%s",
    cluster.Domain.Name,
    cluster.Name,
    queueName)

  return destroyStack(svc, stackName)
}

func destroyMaster(cluster *Cluster, svc *cloudformation.CloudFormation) error {
  stackName := fmt.Sprintf("flight-%s-%s-master", cluster.Domain.Name, cluster.Name)
  return destroyStack(svc, stackName)
}

func destroyClusterNetwork(cluster *Cluster, svc *cloudformation.CloudFormation) error {
  stackName := fmt.Sprintf("flight-%s-%s-network", cluster.Domain.Name, cluster.Name)

  networkStack, err := getStack(svc, stackName)
  if err != nil {
    if aerr, ok := err.(awserr.Error); ok {
      if strings.Contains(aerr.Message(), "does not exist") {
        return nil
      } else {
        return err
      }
    }
  }
  idx, err := strconv.Atoi(getStackTag(networkStack, "flight:network"))
  if err != nil { return err }
  cluster.Network = &ClusterNetwork{idx, networkStack}

  // handle destruction of unassociated NICs
  err = destroyDetachedNICs(cluster.Network.ManagementSubnet())
  if err != nil { return err }

  return destroyStack(svc, stackName)
}

func destroySoloCluster(cluster *Cluster, svc *cloudformation.CloudFormation) error {
  stackName := fmt.Sprintf("flight-cluster-%s", cluster.Name)
  return destroyStack(svc, stackName)
}

func destroyComponent(cluster *Cluster, componentType, componentName string, svc *cloudformation.CloudFormation) error {
  if componentName == "" {
    componentName = componentType
  } else {
    componentName = componentType + "-" + componentName
  }
  stackName := fmt.Sprintf("flight-%s-%s-component-%s", cluster.Domain.Name, cluster.Name, componentName)
  return destroyStack(svc, stackName)
}

func createMaster(cluster *Cluster, svc *cloudformation.CloudFormation) error {
  launchParams := createClusterComponentLaunchParameters(cluster,
    loadParameterSet("cluster-master", ClusterMasterParameters))
  stackName := fmt.Sprintf("flight-%s-%s-master", cluster.Domain.Name, cluster.Name)
  url := TemplateUrl(clusterMasterTemplate)

  tags := cluster.Tags()
  if cluster.ExpiryTime > 0 {
    tags = append(tags, &cloudformation.Tag{Key: aws.String("flight:expiry"), Value: aws.String(strconv.FormatInt(cluster.ExpiryTime, 10))})
  }
  stack, err := createStack(svc, launchParams, tags, url, stackName, "master", cluster.TopicARN, cluster.Domain)
  if err != nil { return err }

  cluster.Master = &Master{stack}
  return nil
}

func createComponent(componentType, componentName, componentParamsFile string, cluster *Cluster, svc *cloudformation.CloudFormation) error {
  launchParams := createClusterComponentLaunchParameters(cluster, loadComponentParameters(componentParamsFile))
  if componentName == "" {
    componentName = componentType
  } else {
    componentName = componentType + "-" + componentName
  }
  stackName := fmt.Sprintf("flight-%s-%s-component-%s", cluster.Domain.Name, cluster.Name, componentName)
  url := TemplateUrl(componentType + ".json")

  _, err := createStack(svc, launchParams, cluster.Tags(), url, stackName, "component", cluster.TopicARN, cluster.Domain)
  if err != nil { return err }

  return nil
}

func loadParameterSet(componentName string, defaultParameterSet map[string]string) map[string]string {
  if Config().ParameterDirectory != "" {
    parameterSet := loadComponentParameters(Config().ParameterDirectory + "/" + componentName + ".yml")
    if len(parameterSet) > 0 {
      return parameterSet
    }
  }
  return defaultParameterSet
}

func createComputeGroup(cluster *Cluster, queueName, queueParamsFile string, expiryTime int64, svc *cloudformation.CloudFormation) error {
  var defaultLaunchParams map[string]string
  if queueParamsFile == "" {
    defaultLaunchParams = loadParameterSet("cluster-compute", ClusterComputeParameters)
  } else {
    defaultLaunchParams = loadComponentParameters(queueParamsFile)
  }
  launchParams := createClusterComponentLaunchParameters(cluster, defaultLaunchParams)
  stackName := fmt.Sprintf("flight-%s-%s-compute-%s",
    cluster.Domain.Name,
    cluster.Name,
    queueName)
  url := TemplateUrl(clusterComputeTemplate)

  tags := cluster.Tags()
  if expiryTime > 0 {
    tags = append(tags, &cloudformation.Tag{Key: aws.String("flight:expiry"), Value: aws.String(strconv.FormatInt(expiryTime, 10))})
  }
  stack, err := createStack(svc, launchParams, tags, url, stackName, "compute", cluster.TopicARN, cluster.Domain)
  if err != nil { return err }

  cluster.ComputeGroups = append(cluster.ComputeGroups, computeGroupFromStack(stack))
  return nil
}

func createClusterNetwork(cluster *Cluster, svc *cloudformation.CloudFormation) error {
  network, err := cluster.Domain.BookNetwork()
  if err != nil { return err }

  cluster.Network = &ClusterNetwork{network, nil}
  launchParams := createClusterComponentLaunchParameters(cluster,
    loadParameterSet("cluster-network", ClusterNetworkParameters))
  stackName := fmt.Sprintf("flight-%s-%s-network", cluster.Domain.Name, cluster.Name)
  url := TemplateUrl(clusterNetworkTemplate)
  tags := append(cluster.Tags(), &cloudformation.Tag{Key: aws.String("flight:network"), Value: aws.String(strconv.Itoa(network))})

  stack, err := createStack(svc, launchParams, tags, url, stackName, "network", cluster.TopicARN, cluster.Domain)
  if err != nil { return err }

  cluster.Network.Stack = stack
  err = cluster.CreateEntity()
  if err != nil { return err }

  return nil
}

func createSoloCluster(cluster *Cluster, svc *cloudformation.CloudFormation) error {
  launchParams := createClusterComponentLaunchParameters(cluster,
    loadParameterSet("solo", SoloParameters))
  stackName := fmt.Sprintf("flight-cluster-%s", cluster.Name)
  url := TemplateUrl(soloClusterTemplate)

  tags := cluster.Tags()
  if cluster.ExpiryTime > 0 {
    tags = append(tags, &cloudformation.Tag{Key: aws.String("flight:expiry"), Value: aws.String(strconv.FormatInt(cluster.ExpiryTime, 10))})
  }
  stack, err := createStack(svc, launchParams, tags, url, stackName, "solo", cluster.TopicARN, nil)
  if err != nil { return err }

  cluster.Master = &Master{stack}
  return nil
}

func createClusterComponentLaunchParameters(cluster *Cluster, parameterSet map[string]string) []*cloudformation.Parameter {
  params := []*cloudformation.Parameter{}
  for key, value := range parameterSet {
    var val string
    switch value  {
    case "%CLUSTER_NAME%":
      val = cluster.Name
    case "%ACCESS_KEY_NAME%":
      val = Config().AccessKeyName
    case "%MASTER_INSTANCE_TYPE%":
      instanceOverride := viper.GetString("master-instance-override")
      if instanceOverride != "" {
        val = "other"
      } else {
        val = viper.GetString("master-instance-type")
      }
    case "%MASTER_INSTANCE_OVERRIDE%":
      val = viper.GetString("master-instance-override")
      if val == "" { val = "%NULL%" }
    case "%MASTER_FEATURES%":
      // XXX - should password-auth be mandated within template?
      masterFeatures := viper.GetString("master-features")
      if masterFeatures != "" {
        val = masterFeatures + " password-auth"
      } else {
        val = "password-auth"
      }
    case "%COMPUTE_INSTANCE_TYPE%":
      instanceOverride := viper.GetString("queue-instance-override")
      if instanceOverride != "" {
        val = "other"
      } else {
        // if we're launching via cluster launch, we use default-queue-instance-type, otherwise we use queue-instance-type
        val = viper.GetString("queue-instance-type")
        if val == "" {
          val = viper.GetString("default-queue-instance-type")
        }
      }
    case "%COMPUTE_INSTANCE_OVERRIDE%":
      val = viper.GetString("queue-instance-override")
      if val == "" { val = "%NULL%" }
    case "%VPC%":
      val = cluster.Domain.VPC()
    case "%NETWORK_POOL%":
      val = cluster.Network.NetworkPool()
    case "%NETWORK_INDEX%":
      val = cluster.Network.NetworkIndex()
    case "%PUB_ROUTE_TABLE%":
      val = cluster.Domain.PublicRouteTable()
    case "%DOMAIN%":
      val = cluster.Domain.Prefix()
    case "%PUB_SUBNET%":
      val = cluster.Network.PublicSubnet()
    case "%MGT_SUBNET%":
      val = cluster.Network.ManagementSubnet()
    case "%PRV_SUBNET%":
      val = cluster.Network.PrivateSubnet()
    case "%PLACEMENT_GROUP%":
      val = cluster.Network.PlacementGroup()
    case "%MASTER_IP%":
      val = cluster.Master.PrivateIP()
    case "%CLUSTER_UUID%":
      val = cluster.Master.ClusterUUID()
    case "%CLUSTER_SECURITY_TOKEN%":
      val = cluster.Master.ClusterSecurityToken()
    default:
      if strings.HasPrefix(value, "%") && strings.HasSuffix(value, "%") {
        configKey := strings.ToLower(strings.Replace(value[1:len(value)-1], "_", "-", -1))
        if viper.IsSet(configKey) {
          val = viper.GetString(configKey)
        } else {
          val = "%NULL%"
        }
      } else {
        val = value
      }
    }
    if val != "%NULL%" {
      // fmt.Println(key + " -> " + val)
      params = append(params, &cloudformation.Parameter{
        ParameterKey: aws.String(key),
        ParameterValue: aws.String(val),
      })
    }
  }
  return params
}

func loadComponentParameters(paramsFile string) map[string]string {
  params := make(map[string]string)
  if paramsFile != "" {
    data, err := ioutil.ReadFile(paramsFile)
    if err != nil { fmt.Println(err.Error()) }
    err = yaml.Unmarshal(data, &params)
    if err != nil { fmt.Println(err.Error()) }
  }
  return params
}

func ExpiredClusters() ([]string, error) {
  stacks, err := ExpiredStacks()
  names := []string{}
  if err != nil { return nil, err }
  for _, stack := range stacks {
    var descriptor string
    name := getStackTag(stack, "flight:cluster")
    stackType := getStackTag(stack, "flight:type")
    domain := getStackTag(stack, "flight:domain")
    if stackType == "master" {
      descriptor = fmt.Sprintf("CLUSTER:%s/%s", domain, name)
    } else if stackType == "compute" {
      var queueName string
      queueNameParts := strings.SplitAfterN(*stack.StackName, "-compute-", 2)
      if len(queueNameParts) > 1 {
        queueName = queueNameParts[1]
      } else {
        queueName = queueNameParts[0]
      }
      descriptor = fmt.Sprintf("QUEUE:%s/%s/%s", domain, name, queueName)
    } else {
      descriptor = fmt.Sprintf("SOLO:%s", name)
    }
    names = append(names, descriptor)
  }
  return names, nil
}
