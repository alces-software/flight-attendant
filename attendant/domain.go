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
  "encoding/xml"
  "fmt"
  "strconv"
  "strings"
  "time"

  "github.com/spf13/viper"

  "github.com/aws/aws-sdk-go/aws"
  "github.com/aws/aws-sdk-go/aws/awserr"
  "github.com/aws/aws-sdk-go/service/cloudformation"
)

var DomainResourceCount int = 35
var DomainPeeringResourceCount int = 4
var DomainPeerRoutesResourceCount int = 2
var DomainVPNResourceCount int = 4
var DomainInternetAccessResourceCount int = 3
var DomainNoInternetAccessResourceCount int = 1

type Domain struct {
  Name string
  Stack *cloudformation.Stack
  MessageHandler func(msg string)
}

type DomainStatus struct {
  Clusters map[string]*Cluster
  Appliances map[string]*Appliance
  HasInternetAccess bool
  VPNConnectionId string
  PeerVPC string
  PeerVPCCIDRBlock string
  VPNDetails VPNConnectionDetails
}

type DomainDetails struct {
  Clusters map[string]*ClusterDetails
  Appliances map[string]*ApplianceDetails
  HasInternetAccess bool
  VPNConnectionId string
  PeerVPC string
  PeerVPCCIDRBlock string
  VPNDetails VPNConnectionDetails
}

type XMLVPNConnection struct {
  Tunnels []struct {
    OutsideClientAddr string `xml:"customer_gateway>tunnel_outside_address>ip_address"`
    InsideClientAddr string `xml:"customer_gateway>tunnel_inside_address>ip_address"`
    InsideClientCidr string `xml:"customer_gateway>tunnel_inside_address>network_cidr"`
    ClientASN string `xml:"customer_gateway>bgp>asn"`

    OutsideAwsAddr string `xml:"vpn_gateway>tunnel_outside_address>ip_address"`
    InsideAwsAddr string `xml:"vpn_gateway>tunnel_inside_address>ip_address"`
    InsideAwsCidr string `xml:"vpn_gateway>tunnel_inside_address>network_cidr"`
    AwsASN string `xml:"vpn_gateway>bgp>asn"`

    SharedKey string `xml:"ike>pre_shared_key"`
  } `xml:"ipsec_tunnel"`
}

type VPNConnectionDetails struct {
  OutsideClientAddr string `yaml:"OutsideClientAddr"`
  ClientASN string `yaml:"ClientASN"`
  InsideCidr string `yaml:"InsideCIDR"`
  Tunnel1 IPSecTunnel
  Tunnel2 IPSecTunnel
}

type IPSecTunnel struct {
  OutsideAwsAddr string `yaml:"OutsideAwsAddr"`
  InsideAwsAddr string `yaml:"InsideAwsAddr"`
  InsideClientAddr string `yaml:"InsideClientAddr"`
  SharedKey string `yaml:"SharedKey"`
  AwsASN string `yaml:"AwsASN"`
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
  return d.getOutput("PubSubnet")
}

func (d *Domain) ManagementSubnet() string {
  return d.getOutput("MgtSubnet")
}

func (d *Domain) PrivateSubnet() string {
  return d.getOutput("PrvSubnet")
}

func (d *Domain) PlacementGroup() string {
  return d.getOutput("PlacementGroup")
}

func (d *Domain) PublicRouteTable() string {
  return d.getOutput("PubRouteTable")
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
  var soloStatus DomainStatus
  soloStatus.Clusters = make(map[string]*Cluster)
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

func (s *DomainStatus) Details() *DomainDetails {
  details := DomainDetails{}
  details.HasInternetAccess = s.HasInternetAccess
  details.VPNConnectionId = s.VPNConnectionId
  details.PeerVPC = s.PeerVPC
  details.PeerVPCCIDRBlock = s.PeerVPCCIDRBlock
  details.VPNDetails = s.VPNDetails
  details.Clusters = make(map[string]*ClusterDetails)
  details.Appliances = make(map[string]*ApplianceDetails)
  for _, cluster := range s.Clusters {
    details.Clusters[cluster.Name] = cluster.Details()
  }
  for _, appliance := range s.Appliances {
    details.Appliances[appliance.Name] = appliance.Details()
  }
  return &details
}

func (d *Domain) Status() (*DomainStatus, error) {
  err := d.AssertExists()
  if err != nil { return nil, err }
  var status DomainStatus
  status.Clusters = make(map[string]*Cluster)
  status.Appliances = make(map[string]*Appliance)
  // check no infrastructure or clusters exist in domain
  err = eachRunningStack(func(stack *cloudformation.Stack) {
    for _, tag := range stack.Tags {
      if *tag.Key == "flight:domain" && *tag.Value == d.Name {
        stackType := getStackTag(stack, "flight:type")
        switch stackType {
        case "master", "network", "compute":
          clusterName := getStackTag(stack, "flight:cluster")
          cluster, exists := status.Clusters[clusterName]
          if ! exists {
            cluster = &Cluster{Name: clusterName, Domain: d, ExpiryTime: -1}
            status.Clusters[clusterName] = cluster
          }
          if stackType == "master" {
            cluster.Master = &Master{stack}
          } else if stackType == "network" {
            idx, _ := strconv.Atoi(getStackTag(stack,"flight:network"))
            cluster.Network = &ClusterNetwork{idx, stack}
          } else if stackType == "compute" {
            cluster.ComputeGroups = append(cluster.ComputeGroups, computeGroupFromStack(stack))
          }
        case "appliance":
          applianceName := getStackTag(stack, "flight:appliance")
          status.Appliances[applianceName] = &Appliance{applianceName, d, stack, nil}
        }
      }
    }
  })
  status.HasInternetAccess = d.HasInternetAccess()
  status.VPNConnectionId = getStackOutput(d.Stack, "VpnConnection")
  status.PeerVPC = getStackParameter(d.Stack, "PeerVPC")
  status.PeerVPCCIDRBlock = getStackParameter(d.Stack, "PeerVPCCIDRBlock")
  if status.VPNConnectionId != "" {
    status.VPNDetails, err = getVPNDetails(status.VPNConnectionId)
  }
  return &status, err
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

func (d *Domain) Create(prefix string, domainParamsFile string) error {
  var defaultLaunchParams map[string]string
  if domainParamsFile == "" {
    defaultLaunchParams = loadParameterSet("domain", DomainParameters)
  } else {
    defaultLaunchParams = loadComponentParameters(domainParamsFile)
  }

  d.MessageHandler(fmt.Sprintf("COUNTERS=%d",resourceCountFor(defaultLaunchParams)))

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
    Parameters: createDomainLaunchParameters(d, defaultLaunchParams),
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

func (d *Domain) HasInternetAccess() bool {
  return getStackParameter(d.Stack, "AllowInternetAccess") != "0"
}

func (d *Domain) MasterIP() string {
  // get controller stack
  a := NewAppliance("controller", d, nil)
  a.LoadStack()
  if a.Stack == nil {
    return ""
  } else {
    return a.Details().Extra["PrivateIpAddress"]
  }
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

func createDomainLaunchParameters(domain *Domain, parameterSet map[string]string) []*cloudformation.Parameter {
  params := []*cloudformation.Parameter{}
  for key, value := range parameterSet {
    var val string
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

func resourceCountFor(params map[string]string) int {
  resourceCount := DomainResourceCount

  peerVpc := params["PeerVPC"]
  if peerVpc == "%PEER_VPC%" {
    peerVpc = viper.GetString("peer-vpc")
  }
  if peerVpc != "" {
    resourceCount += DomainPeeringResourceCount
    peerVpcRouteTable := params["PeerVPCRouteTable"]
    if peerVpcRouteTable == "%PEER_VPCROUTE_TABLE%" {
      peerVpcRouteTable = viper.GetString("peer-vpc-route-table")
    }
    if peerVpcRouteTable != "" {
      resourceCount += DomainPeerRoutesResourceCount
    }
  }

  allowInternet := params["AllowInternetAccess"]
  if allowInternet == "%ALLOW_INTERNET_ACCCESS%" {
    allowInternet = viper.GetString("allow-internet-access")
  }
  if allowInternet == "0" {
    resourceCount -= DomainInternetAccessResourceCount
    resourceCount += DomainNoInternetAccessResourceCount
  }

  vpnGateway := params["VPNCustomerGateway"]
  if vpnGateway == "%VPN_CUSTOMER_GATEWAY%" {
    vpnGateway = viper.GetString("vpn-customer-gateway")
  }
  if vpnGateway != "" {
    resourceCount += DomainVPNResourceCount
  }

  return resourceCount
}

func getVPNDetails(vpnConnectionId string) (VPNConnectionDetails, error) {
  var details VPNConnectionDetails
  var xmlData XMLVPNConnection
  xmlStr, err := describeVPNConnection(vpnConnectionId)
  if err != nil { return details, err }
  xml.Unmarshal([]byte(*xmlStr), &xmlData)
  if len(xmlData.Tunnels) == 0 { return details, fmt.Errorf("Unable to parse VPN connection data") }
  details.OutsideClientAddr = xmlData.Tunnels[0].OutsideClientAddr
  details.ClientASN = xmlData.Tunnels[0].ClientASN
  details.InsideCidr = xmlData.Tunnels[0].InsideAwsCidr
  details.Tunnel1 = IPSecTunnel{
    xmlData.Tunnels[0].OutsideAwsAddr,
    xmlData.Tunnels[0].InsideAwsAddr,
    xmlData.Tunnels[0].InsideClientAddr,
    xmlData.Tunnels[0].SharedKey,
    xmlData.Tunnels[0].AwsASN,
  }
  details.Tunnel2 = IPSecTunnel{
    xmlData.Tunnels[1].OutsideAwsAddr,
    xmlData.Tunnels[1].InsideAwsAddr,
    xmlData.Tunnels[1].InsideClientAddr,
    xmlData.Tunnels[1].SharedKey,
    xmlData.Tunnels[1].AwsASN,
  }

  return details, nil
}
