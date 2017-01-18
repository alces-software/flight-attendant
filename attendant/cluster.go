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
  "io/ioutil"
  "strconv"
  "strings"
  "time"

	"github.com/spf13/viper"

  "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"

  "gopkg.in/yaml.v2"
)

var clusterNetworkTemplate = "cluster-network.json"
var clusterMasterTemplate = "cluster-master.json"
var clusterComputeTemplate = "cluster-compute.json"
var soloClusterTemplate = "solo-cluster.json"

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
var SoloClusterResourceCount int = 48

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
  return getStackParameter(c.Stack, "NetworkingPool")
}

func (c *ClusterNetwork) NetworkIndex() string {
  return getStackParameter(c.Stack, "NetworkingIndex")
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
    // create compute group(s)
    err = createComputeGroup(c, svc)
    if err != nil { return err }
  }

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
  }

  return nil
}

func (c *Cluster) GetDetails() string {
  if c.Master != nil {
    ip := getStackOutput(c.Master.Stack, "AccessIP")
    keypair := getStackParameter(c.Master.Stack, "AccessKeyName")
    username := getStackOutput(c.Master.Stack, "Username")
    url := getStackOutput(c.Master.Stack, "WebAccess")
    componentStacks, _ := getComponentStacksForCluster(c)
    uuid := getStackConfigValue(c.Master.Stack, "UUID")
    if uuid == "" { uuid = "<unknown>" }
    token := getStackConfigValue(c.Master.Stack, "Token")
    if token == "" { token = "<unknown>" }
    details := fmt.Sprintf("Administrator username: %s\nIP address: %s\nKey pair: %s\nAccess URL: %s\nUUID: %s\nToken: %s\n", ip, keypair, username, url, uuid, token)
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
  launchParams := createMasterLaunchParameters(cluster)
  stackName := fmt.Sprintf("flight-%s-%s-master", cluster.Domain.Name, cluster.Name)
  url := TemplateUrl(clusterMasterTemplate)

  stack, err := createStack(svc, launchParams, cluster.Tags(), url, stackName, "master", cluster.TopicARN, cluster.Domain)
  if err != nil { return err }

  cluster.Master = &Master{stack}
  return nil
}

func createComponent(componentType, componentName, componentParamsFile string, cluster *Cluster, svc *cloudformation.CloudFormation) error {
  launchParams := createComponentLaunchParameters(cluster, componentParamsFile)
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

func createComputeGroup(cluster *Cluster, svc *cloudformation.CloudFormation) error {
  launchParams := createComputeLaunchParameters(cluster)
  stackName := fmt.Sprintf("flight-%s-%s-compute-%d",
    cluster.Domain.Name,
    cluster.Name,
    len(cluster.ComputeGroups) + 1)
  url := TemplateUrl(clusterComputeTemplate)

  stack, err := createStack(svc, launchParams, cluster.Tags(), url, stackName, "compute", cluster.TopicARN, cluster.Domain)
  if err != nil { return err }

  cluster.ComputeGroups = append(cluster.ComputeGroups, &ComputeGroup{stack})
  return nil
}

func createClusterNetwork(cluster *Cluster, svc *cloudformation.CloudFormation) error {
  network, err := cluster.Domain.BookNetwork()
  if err != nil { return err }

  launchParams := createNetworkLaunchParameters(cluster, network)
  stackName := fmt.Sprintf("flight-%s-%s-network", cluster.Domain.Name, cluster.Name)
  url := TemplateUrl(clusterNetworkTemplate)
  tags := append(cluster.Tags(), &cloudformation.Tag{Key: aws.String("flight:network"), Value: aws.String(strconv.Itoa(network))})

  stack, err := createStack(svc, launchParams, tags, url, stackName, "network", cluster.TopicARN, cluster.Domain)
  if err != nil { return err }

  cluster.Network = &ClusterNetwork{network, stack}
  err = cluster.CreateEntity()
  if err != nil { return err }

  return nil
}

func createSoloCluster(cluster *Cluster, svc *cloudformation.CloudFormation) error {
  launchParams := createSoloLaunchParameters(cluster)
  stackName := fmt.Sprintf("flight-cluster-%s", cluster.Name)
  url := TemplateUrl(soloClusterTemplate)

  stack, err := createStack(svc, launchParams, cluster.Tags(), url, stackName, "solo", cluster.TopicARN, nil)
  if err != nil { return err }

  cluster.Master = &Master{stack}
  return nil
}

func createNetworkLaunchParameters(cluster *Cluster, network int) []*cloudformation.Parameter {
  networkPool := (network / 32) + 1
  networkIndex := (network % 32) + 1

  params := []*cloudformation.Parameter{
    {
      ParameterKey: aws.String("FlightVPC"),
      ParameterValue: aws.String(cluster.Domain.VPC()),
    },
    {
      ParameterKey: aws.String("NetworkingPool"),
      ParameterValue: aws.String(strconv.Itoa(networkPool)),
    },
    {
      ParameterKey: aws.String("NetworkingIndex"),
      ParameterValue: aws.String(strconv.Itoa(networkIndex)),
    },
    {
      ParameterKey: aws.String("PubRouteTable"),
      ParameterValue: aws.String(cluster.Domain.PublicRouteTable()),
    },
  }
  return params
}

func createMasterLaunchParameters(cluster *Cluster) []*cloudformation.Parameter {
  masterFeatures := viper.GetString("master-features")
  if masterFeatures != "" {
    masterFeatures = masterFeatures + " password-auth"
  } else {
    masterFeatures = "password-auth"
  }
  params := []*cloudformation.Parameter{
    {
      ParameterKey: aws.String("AccessKeyName"),
      ParameterValue: aws.String(Config().AccessKeyName),
    },
    {
      ParameterKey: aws.String("AccessNetwork"),
      ParameterValue: aws.String(viper.GetString("access-network")),
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
      ParameterKey: aws.String("Domain"),
      ParameterValue: aws.String(cluster.Domain.Prefix()),
    },
    {
      ParameterKey: aws.String("NetworkingPool"),
      ParameterValue: aws.String(cluster.Network.NetworkPool()),
    },
    {
      ParameterKey: aws.String("NetworkingIndex"),
      ParameterValue: aws.String(cluster.Network.NetworkIndex()),
    },
    {
      ParameterKey: aws.String("PubSubnet"),
      ParameterValue: aws.String(cluster.Network.PublicSubnet()),
    },
    {
      ParameterKey: aws.String("MgtSubnet"),
      ParameterValue: aws.String(cluster.Network.ManagementSubnet()),
    },
    {
      ParameterKey: aws.String("PlacementGroup"),
      ParameterValue: aws.String(cluster.Network.PlacementGroup()),
    },
    {
      ParameterKey: aws.String("PrvSubnet"),
      ParameterValue: aws.String(cluster.Network.PrivateSubnet()),
    },
    {
      ParameterKey: aws.String("FlightFeatures"),
      ParameterValue: aws.String(masterFeatures),
    },
    {
      ParameterKey: aws.String("FlightProfileBucket"),
      ParameterValue: aws.String(viper.GetString("profile-bucket")),
    },
    {
      ParameterKey: aws.String("FlightProfiles"),
      ParameterValue: aws.String(viper.GetString("master-profiles")),
    },
    {
      ParameterKey: aws.String("PreloadSoftware"),
      ParameterValue: aws.String(viper.GetString("preload-software")),
    },
    {
      ParameterKey: aws.String("SchedulerType"),
      ParameterValue: aws.String(viper.GetString("scheduler-type")),
    },
    {
      ParameterKey: aws.String("VolumeLayout"),
      ParameterValue: aws.String(viper.GetString("master-volume-layout")),
    },
    {
      ParameterKey: aws.String("VolumeEncryptionPolicy"),
      ParameterValue: aws.String(viper.GetString("master-volume-encryption-policy")),
    },
    {
      ParameterKey: aws.String("MasterSystemVolumeSize"),
      ParameterValue: aws.String(viper.GetString("master-system-volume-size")),
    },
    {
      ParameterKey: aws.String("MasterSystemVolumeType"),
      ParameterValue: aws.String(viper.GetString("master-system-volume-type")),
    },
    {
      ParameterKey: aws.String("AppsVolumeSize"),
      ParameterValue: aws.String(viper.GetString("master-apps-volume-size")),
    },
    {
      ParameterKey: aws.String("AppsVolumeType"),
      ParameterValue: aws.String(viper.GetString("master-apps-volume-type")),
    },
    {
      ParameterKey: aws.String("HomeVolumeSize"),
      ParameterValue: aws.String(viper.GetString("master-home-volume-size")),
    },
    {
      ParameterKey: aws.String("HomeVolumeType"),
      ParameterValue: aws.String(viper.GetString("master-home-volume-type")),
    },
  }
  instanceType := viper.GetString("master-instance-override")
  if instanceType != "" {
    params = append(params, []*cloudformation.Parameter{
      {
        ParameterKey: aws.String("MasterInstanceType"),
        ParameterValue: aws.String("other"),
      },
      {
        ParameterKey: aws.String("MasterInstanceTypeOther"),
        ParameterValue: aws.String(instanceType),
      },
    }...)
  } else {
    params = append(params, &cloudformation.Parameter{
      ParameterKey: aws.String("MasterInstanceType"),
      ParameterValue: aws.String(viper.GetString("master-instance-type")),
    })
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

func createComponentLaunchParameters(cluster *Cluster, paramsFile string) []*cloudformation.Parameter {
  params := []*cloudformation.Parameter{}
  for key, value := range loadComponentParameters(paramsFile) {
    var val string
    switch value  {
    case "%ACCESS_KEY_NAME%":
      val = Config().AccessKeyName
    case "%ACCESS_NETWORK%":
      val = viper.GetString("access-network")
    case "%ACCESS_USERNAME%":
      val = viper.GetString("admin-user-name")
    case "%CLUSTER_NAME%":
      val = cluster.Name
    case "%VPC%":
      val = cluster.Domain.VPC()
    case "%DOMAIN%":
      val = cluster.Domain.Prefix()
    case "%NETWORK_POOL%":
      val = cluster.Network.NetworkPool()
    case "%NETWORK_INDEX%":
      val = cluster.Network.NetworkIndex()
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
    case "%COMPUTE_PROFILES%":
      val = viper.GetString("compute-profiles")
    case "%PROFILE_BUCKET%":
      val = viper.GetString("profile-bucket")
    case "%COMPUTE_FEATURES%":
      val = viper.GetString("compute-features")
    case "%SCHEDULER_TYPE%":
      val = viper.GetString("scheduler-type")
    default:
      val = value
    }
    params = append(params, &cloudformation.Parameter{
      ParameterKey: aws.String(key),
      ParameterValue: aws.String(val),
    })
  }
  return params
}

func createComputeLaunchParameters(cluster *Cluster) []*cloudformation.Parameter {
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
      ParameterKey: aws.String("Domain"),
      ParameterValue: aws.String(cluster.Domain.Prefix()),
    },
    {
      ParameterKey: aws.String("NetworkingPool"),
      ParameterValue: aws.String(cluster.Network.NetworkPool()),
    },
    {
      ParameterKey: aws.String("NetworkingIndex"),
      ParameterValue: aws.String(cluster.Network.NetworkIndex()),
    },
    {
      ParameterKey: aws.String("SchedulerType"),
      ParameterValue: aws.String(viper.GetString("scheduler-type")),
    },
    {
      ParameterKey: aws.String("ComputeSpotPrice"),
      ParameterValue: aws.String(viper.GetString("compute-spot-price")),
    },
    {
      ParameterKey: aws.String("AutoscalingPolicy"),
      ParameterValue: aws.String(viper.GetString("compute-autoscaling-policy")),
    },
    {
      ParameterKey: aws.String("ComputeInitialNodes"),
      ParameterValue: aws.String(viper.GetString("compute-initial-nodes")),
    },
    {
      ParameterKey: aws.String("PrvSubnet"),
      ParameterValue: aws.String(cluster.Network.PrivateSubnet()),
    },
    {
      ParameterKey: aws.String("MgtSubnet"),
      ParameterValue: aws.String(cluster.Network.ManagementSubnet()),
    },
    {
      ParameterKey: aws.String("PlacementGroup"),
      ParameterValue: aws.String(cluster.Network.PlacementGroup()),
    },
    {
      ParameterKey: aws.String("FlightFeatures"),
      ParameterValue: aws.String(viper.GetString("compute-features")),
    },
    {
      ParameterKey: aws.String("FlightProfileBucket"),
      ParameterValue: aws.String(viper.GetString("profile-bucket")),
    },
    {
      ParameterKey: aws.String("FlightProfiles"),
      ParameterValue: aws.String(viper.GetString("compute-profiles")),
    },
    {
      ParameterKey: aws.String("MasterPrivateIP"),
      ParameterValue: aws.String(cluster.Master.PrivateIP()),
    },
    {
      ParameterKey: aws.String("ClusterUUID"),
      ParameterValue: aws.String(cluster.Master.ClusterUUID()),
    },
    {
      ParameterKey: aws.String("ClusterSecurityToken"),
      ParameterValue: aws.String(cluster.Master.ClusterSecurityToken()),
    },
  }
  instanceType := viper.GetString("compute-instance-override")
  if instanceType != "" {
    params = append(params, []*cloudformation.Parameter{
      {
        ParameterKey: aws.String("ComputeInstanceType"),
        ParameterValue: aws.String("other"),
      },
      {
        ParameterKey: aws.String("ComputeInstanceTypeOther"),
        ParameterValue: aws.String(instanceType),
      },
    }...)
  } else {
    params = append(params, &cloudformation.Parameter{
      ParameterKey: aws.String("ComputeInstanceType"),
      ParameterValue: aws.String(viper.GetString("compute-instance-type")),
    })
  }
  return params
}

func createSoloLaunchParameters(cluster *Cluster) []*cloudformation.Parameter {
  masterFeatures := viper.GetString("master-features")
  params := []*cloudformation.Parameter{
    {
      ParameterKey: aws.String("AccessKeyName"),
      ParameterValue: aws.String(Config().AccessKeyName),
    },
    {
      ParameterKey: aws.String("AccessNetwork"),
      ParameterValue: aws.String(viper.GetString("access-network")),
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
      ParameterKey: aws.String("FlightFeatures"),
      ParameterValue: aws.String(masterFeatures),
    },
    {
      ParameterKey: aws.String("FlightProfileBucket"),
      ParameterValue: aws.String(viper.GetString("profile-bucket")),
    },
    {
      ParameterKey: aws.String("FlightProfiles"),
      ParameterValue: aws.String(viper.GetString("master-profiles")),
    },
    {
      ParameterKey: aws.String("PreloadSoftware"),
      ParameterValue: aws.String(viper.GetString("preload-software")),
    },
    {
      ParameterKey: aws.String("SchedulerType"),
      ParameterValue: aws.String(viper.GetString("scheduler-type")),
    },
    {
      ParameterKey: aws.String("VolumeLayout"),
      ParameterValue: aws.String(viper.GetString("master-volume-layout")),
    },
    {
      ParameterKey: aws.String("VolumeEncryptionPolicy"),
      ParameterValue: aws.String(viper.GetString("master-volume-encryption-policy")),
    },
    {
      ParameterKey: aws.String("MasterSystemVolumeSize"),
      ParameterValue: aws.String(viper.GetString("master-system-volume-size")),
    },
    {
      ParameterKey: aws.String("MasterSystemVolumeType"),
      ParameterValue: aws.String(viper.GetString("master-system-volume-type")),
    },
    {
      ParameterKey: aws.String("AppsVolumeSize"),
      ParameterValue: aws.String(viper.GetString("master-apps-volume-size")),
    },
    {
      ParameterKey: aws.String("AppsVolumeType"),
      ParameterValue: aws.String(viper.GetString("master-apps-volume-type")),
    },
    {
      ParameterKey: aws.String("HomeVolumeSize"),
      ParameterValue: aws.String(viper.GetString("master-home-volume-size")),
    },
    {
      ParameterKey: aws.String("HomeVolumeType"),
      ParameterValue: aws.String(viper.GetString("master-home-volume-type")),
    },
    {
      ParameterKey: aws.String("ComputeSpotPrice"),
      ParameterValue: aws.String(viper.GetString("compute-spot-price")),
    },
    {
      ParameterKey: aws.String("AutoscalingPolicy"),
      ParameterValue: aws.String(viper.GetString("compute-autoscaling-policy")),
    },
    {
      ParameterKey: aws.String("ComputeInitialNodes"),
      ParameterValue: aws.String(viper.GetString("compute-initial-nodes")),
    },
  }
  instanceType := viper.GetString("master-instance-override")
  if instanceType != "" {
    params = append(params, []*cloudformation.Parameter{
      {
        ParameterKey: aws.String("MasterInstanceType"),
        ParameterValue: aws.String("other"),
      },
      {
        ParameterKey: aws.String("MasterInstanceTypeOther"),
        ParameterValue: aws.String(instanceType),
      },
    }...)
  } else {
    params = append(params, &cloudformation.Parameter{
      ParameterKey: aws.String("MasterInstanceType"),
      ParameterValue: aws.String(viper.GetString("master-instance-type")),
    })
  }
  instanceType = viper.GetString("compute-instance-override")
  if instanceType != "" {
    params = append(params, []*cloudformation.Parameter{
      {
        ParameterKey: aws.String("ComputeInstanceType"),
        ParameterValue: aws.String("other"),
      },
      {
        ParameterKey: aws.String("ComputeInstanceTypeOther"),
        ParameterValue: aws.String(instanceType),
      },
    }...)
  } else {
    params = append(params, &cloudformation.Parameter{
      ParameterKey: aws.String("ComputeInstanceType"),
      ParameterValue: aws.String(viper.GetString("compute-instance-type")),
    })
  }
  return params
}
