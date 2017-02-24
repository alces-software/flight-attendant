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
  "time"

  "github.com/spf13/viper"

  "github.com/aws/aws-sdk-go/aws"
  "github.com/aws/aws-sdk-go/service/cloudformation"
)

type Appliance struct {
  Name string
  Domain *Domain
  Stack *cloudformation.Stack
  MessageHandler func(msg string)
}

type ApplianceDetails struct {
  Ip string
  KeyPair string
  Url string
  Extra map[string]string
}

var ApplianceTemplates = map[string]string{
  "directory": "directory.json",
  "storage-manager": "storage-manager.json",
  "access-manager": "access-manager.json",
  "monitor": "monitor.json",
  "controller": "controller.json",
  "silo": "silo.json",
}

var ApplianceResourceCounts = map[string]int {
  "directory": 11,
  "storage-manager": 9,
  "access-manager": 9,
  "monitor": 11,
  "controller": 16,
  "silo": 10,
}

var ApplianceInstanceTypes = []string{
  "small-t2.large",
  "small-c3.large",
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

func (a *Appliance) LoadStack() error {
  if a.Stack != nil {
    return nil
  }
  svc, err := CloudFormation()
  if err != nil { return err }
  stack, err := getStack(svc, "flight-" + a.Domain.Name + "-" + a.Name)
  if err != nil { return err }
  a.Stack = stack
  return nil
}

func (a *Appliance) Create() error {
  svc, err := CloudFormation()
  if err != nil { return err }

  url := TemplateUrl(ApplianceTemplates[a.Name])
  if url == "" {
    return fmt.Errorf("Unknown appliance type: %s", a.Name)
  }

  if err = a.Domain.AssertReady(); err != nil { return err }

  var launchParams []*cloudformation.Parameter
  switch a.Name {
  case "directory", "monitor":
    launchParams = createApplianceLaunchParameters(a, loadParameterSet(a.Name, DomainApplianceParameters))
  case "controller":
    defaultParams := make(map[string]string)
    for k,v := range DomainApplianceParameters {
      defaultParams[k] = v
    }
    defaultParams["PrvSubnet"] = "%PRV_SUBNET%"
    launchParams = createApplianceLaunchParameters(a, loadParameterSet(a.Name, defaultParams))
  case "silo":
    launchParams = createApplianceLaunchParameters(a, loadParameterSet(a.Name, SiloParameters))
  case "access-manager", "storage-manager":
    launchParams = createApplianceLaunchParameters(a, loadParameterSet(a.Name, BasicApplianceParameters))
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

func (a Appliance) Purge() error {
  svc, err := CloudFormation()
  if err != nil { return err }

  stackName := fmt.Sprintf("flight-%s-%s", a.Domain.Name, a.Name)
  if err != nil { return err }

  err = destroyStack(svc, stackName)
  if err != nil { return err }

  err = cleanupEventHandling(stackName)
  if err != nil { return err }

  return err
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

func (a Appliance) Details() *ApplianceDetails {
  var details ApplianceDetails = ApplianceDetails{}
  details.Extra = make(map[string]string)
  switch a.Name {
  case "directory":
    details.Ip = getStackOutput(a.Stack, "DirectoryAccessIP")
    details.KeyPair = getStackParameter(a.Stack, "AccessKeyName")
    details.Url = getStackOutput(a.Stack, "DirectoryWebAccess")
    configData := strings.Split(getStackOutput(a.Stack, "ConfigurationResult"), "\"")
    var otherData []string
    if len(configData) > 3 {
      otherData = strings.Split(configData[3],";")
    }
    for _, otherDatum := range otherData {
      keyVal := strings.Split(strings.TrimSpace(otherDatum), ":")
      if len(keyVal) > 1 {
        details.Extra[keyVal[0]] = details.Extra[keyVal[1]]
      }
    }
  case "controller":
    details.Ip = getStackOutput(a.Stack, "ControllerAccessIP")
    details.KeyPair = getStackParameter(a.Stack, "AccessKeyName")
    details.Url = getStackOutput(a.Stack, "ControllerWebAccess")
    configData := strings.Split(getStackOutput(a.Stack, "ConfigurationResult"), "\"")
    var otherData []string
    if len(configData) > 3 {
      otherData = strings.Split(configData[3],";")
    }
    for _, otherDatum := range otherData {
      keyVal := strings.Split(strings.TrimSpace(otherDatum), ":")
      if len(keyVal) > 1 {
        details.Extra[keyVal[0]] = details.Extra[keyVal[1]]
      }
    }
    details.Extra["PrivateIpAddress"] = getStackOutput(a.Stack, "ControllerPrivateIP")
  case "monitor":
    details.Ip = getStackOutput(a.Stack, "MonitorAccessIP")
    details.KeyPair = getStackParameter(a.Stack, "AccessKeyName")
    details.Url = getStackOutput(a.Stack, "MonitorWebAccess")
    configData := strings.Split(getStackOutput(a.Stack, "ConfigurationResult"), "\"")
    var otherData []string
    if len(configData) > 3 {
      otherData = strings.Split(configData[3],";")
    }
    for _, otherDatum := range otherData {
      keyVal := strings.Split(strings.TrimSpace(otherDatum), ":")
      if len(keyVal) > 1 {
        details.Extra[keyVal[0]] = strings.TrimSpace(keyVal[1])
      }
    }
  case "storage-manager":
    details.Url = getStackOutput(a.Stack, "StorageManagerWebAccess")
  case "access-manager":
    details.Url = getStackOutput(a.Stack, "AccessManagerWebAccess")
  case "silo":
    details.KeyPair = getStackParameter(a.Stack, "AccessKeyName")
  }
  return &details
}

func (a Appliance) GetDetails() string {
  var details string
  switch a.Name {
  case "directory":
    ip := getStackOutput(a.Stack, "DirectoryAccessIP")
    keypair := getStackParameter(a.Stack, "AccessKeyName")
    url := getStackOutput(a.Stack, "DirectoryWebAccess")
    configData := strings.Split(getStackOutput(a.Stack, "ConfigurationResult"), "\"")
    var otherData []string
    if len(configData) > 3 {
      otherData = strings.Split(configData[3],";")
    }
    otherDetails := ""
    for _, otherDatum := range otherData {
      otherDetails += strings.TrimSpace(otherDatum) + "\n"
    }
    details = fmt.Sprintf("IP address: %s\nKey pair: %s\nAccess URL: %s\n%s", ip, keypair, url, otherDetails)
  case "controller":
    ip := getStackOutput(a.Stack, "ControllerAccessIP")
    keypair := getStackParameter(a.Stack, "AccessKeyName")
    url := getStackOutput(a.Stack, "ControllerWebAccess")
    configData := strings.Split(getStackOutput(a.Stack, "ConfigurationResult"), "\"")
    var otherData []string
    if len(configData) > 3 {
      otherData = strings.Split(configData[3],";")
    }
    otherDetails := ""
    for _, otherDatum := range otherData {
      otherDetails += strings.TrimSpace(otherDatum) + "\n"
    }
    details = fmt.Sprintf("IP address: %s\nKey pair: %s\nAccess URL: %s\n%s", ip, keypair, url, otherDetails)
  case "monitor":
    ip := getStackOutput(a.Stack, "MonitorAccessIP")
    keypair := getStackParameter(a.Stack, "AccessKeyName")
    url := getStackOutput(a.Stack, "MonitorWebAccess")
    configData := strings.Split(getStackOutput(a.Stack, "ConfigurationResult"), "\"")
    var otherData []string
    if len(configData) > 3 {
      otherData = strings.Split(configData[3],";")
    }
    otherDetails := ""
    for _, otherDatum := range otherData {
      otherDetails += strings.TrimSpace(otherDatum) + "\n"
    }
    details = fmt.Sprintf("IP address: %s\nKey pair: %s\nAccess URL: %s\n%s", ip, keypair, url, otherDetails)
  case "storage-manager":
    url := getStackOutput(a.Stack, "StorageManagerWebAccess")
    details = fmt.Sprintf("Access URL: %s\n", url)
  case "access-manager":
    url := getStackOutput(a.Stack, "AccessManagerWebAccess")
    details = fmt.Sprintf("Access URL: %s\n", url)
  case "silo":
    keypair := getStackParameter(a.Stack, "AccessKeyName")
    details = fmt.Sprintf("Key pair: %s\n", keypair)
  }
  return details
}

func createApplianceLaunchParameters(appliance *Appliance, parameterSet map[string]string) []*cloudformation.Parameter {
  params := []*cloudformation.Parameter{}
  for key, value := range parameterSet {
    var val string
    switch value  {
    case "%ACCESS_KEY_NAME%":
      val = Config().AccessKeyName
    case "%VPC%":
      val = appliance.Domain.VPC()
    case "%DOMAIN%":
      val = appliance.Domain.Prefix()
    case "%PUB_SUBNET%":
      val = appliance.Domain.PublicSubnet()
    case "%MGT_SUBNET%":
      val = appliance.Domain.ManagementSubnet()
    case "%PRV_SUBNET%":
      val = appliance.Domain.PrivateSubnet()
    case "%PLACEMENT_GROUP%":
      val = appliance.Domain.PlacementGroup()
    case "%APPLIANCE_FEATURES%":
      val = viper.GetString(appliance.Name + "-features")
    case "%APPLIANCE_PROFILES%":
      val = viper.GetString(appliance.Name + "-profiles")
    case "%APPLIANCE_INSTANCE_TYPE%":
      val = viper.GetString(appliance.Name + "-instance-type")
      if val == "" { val = viper.GetString("appliance-instance-type") }
      if val == "" { val = ApplianceInstanceTypes[0] }
    case "%MASTER_IP%":
      val = appliance.Domain.MasterIP()
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
      params = append(params, &cloudformation.Parameter{
        ParameterKey: aws.String(key),
        ParameterValue: aws.String(val),
      })
    }
  }
  return params
}
