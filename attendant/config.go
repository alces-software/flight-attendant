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
  "os"
  "strings"
  "gopkg.in/yaml.v2"
)

var DefaultTemplateRoot = "https://s3-eu-west-1.amazonaws.com/alces-flight/Templates"

var ConfigDefaults = map[string]string {
  "region": "us-east-1",
  "access-key": "",
  "secret-key": "",
  "template-root": "",
  "template-set": FlightRelease,
  "parameter-directory": "",
  
  "admin-user-name": "alces",
  "access-network": "0.0.0.0/0",
  "scheduler-type": "gridscheduler",
  "profile-bucket": "",

  "appliance-instance-type": "",

  "directory-profiles": "",
  "directory-features": "",
  "directory-instance-type": "",

  "monitor-profiles": "",
  "monitor-features": "",
  "monitor-instance-type": "",

  "access-manager-profiles": "",
  "access-manager-features": "",
  "access-manager-instance-type": "",

  "storage-manager-profiles": "",
  "storage-manager-features": "",
  "storage-manager-instance-type": "",

  "master-profiles": "",
  "master-features": "",
  "master-instance-type": MasterInstanceTypes[0],
  "master-instance-override": "",
  "preload-software": "-none-",
  "master-volume-layout": "standard",
  "master-volume-encryption-policy": "unencrypted",
  "master-system-volume-size": "500",
  "master-system-volume-type": "magnetic.standard",
  "master-home-volume-size": "400",
  "master-home-volume-type": "magnetic.standard",
  "master-apps-volume-size": "100",
  "master-apps-volume-type": "magnetic.standard",

  "compute-profiles": "",
  "compute-features": "",
  "default-queue-instance-type": ComputeInstanceTypes[0],
  "queue-instance-type": "",
  "queue-instance-override": "",
  "compute-spot-price": "0.5",
  "compute-autoscaling-policy": "enabled",
  "compute-group-label": "",
  "compute-initial-nodes": "1",
  "compute-system-volume-type": "magnetic.standard",

  "peer-vpc": "",
  "peer-vpc-route-table": "",
  "peer-vpc-cidr-block": "",
  "vpn-customer-gateway": "",

  "allow-internet-access": "1",
}

type Configuration struct {
  AwsRegion string
  AwsAccessKey string
  AwsSecretKey string
  AccessKeyName string
  TemplateRoot string
  TemplateSet string
  ParameterDirectory string
}

var config *Configuration

func Config() *Configuration {
  if config != nil {
    return config
  }
  config = &Configuration{
    AwsRegion: "us-east-1",
    AccessKeyName: "flight-admin",
    TemplateRoot: DefaultTemplateRoot,
    TemplateSet: FlightRelease,
  }
  return config
}

func (c *Configuration) IsValidKeyPair() bool {
  return IsValidKeyPairName(c.AccessKeyName)
}

func RenderConfig() ([]byte, error) {
  return yaml.Marshal(&ConfigDefaults)
}

func RenderConfigValues() (string, error) {
  s := ""
  s += fmt.Sprintf(" == compute instance type ==\n\n     %s\n\n", strings.Join(ComputeInstanceTypes, "\n     "))
  s += fmt.Sprintf(" == master instance type ==\n\n     %s\n\n", strings.Join(MasterInstanceTypes, "\n     "))
  s += fmt.Sprintf(" == appliance instance type ==\n\n     %s\n\n", strings.Join(ApplianceInstanceTypes, "\n     "))
  s += fmt.Sprintf(" == instance type override ==\n\n     %s\n\n", strings.Join(InstanceTypes, "\n     "))
  s += fmt.Sprintf(" == system volume type ==\n\n     %s\n\n", strings.Join(SystemVolumeTypes, "\n     "))
  s += fmt.Sprintf(" == other volume type ==\n\n     %s\n\n", strings.Join(OtherVolumeTypes, "\n     "))
  s += fmt.Sprintf(" == preload software ==\n\n     %s\n\n", strings.Join(SoftwareTypes, "\n     "))
  s += fmt.Sprintf(" == scheduler type ==\n\n     %s\n\n", strings.Join(SchedulerTypes, "\n     "))
  return s, nil
}

func TemplateUrl(templateName string) string {
  templateSet := Config().TemplateSet
  var url string
  if templateSet == "" {
    url = Config().TemplateRoot + "/" + templateName
  } else {
    url = Config().TemplateRoot + "/" + Config().TemplateSet + "/" + templateName
  }
  return url
}

func CreateParameterDirectory(directory string) error {
  // Make directory
  err := os.Mkdir(directory, 0755)
  if err != nil { return err }
  for name, parameterSet := range ParameterSets {
    f, err := os.Create(directory + "/" + name + ".yml")
    if err != nil { return err }
    yaml, err := yaml.Marshal(parameterSet)
    if err != nil { return err }
    _, err = f.Write(yaml)
    if err != nil { return err }
    fmt.Println("Wrote: " + directory + "/" + name + ".yml")
  }
  return nil
}
