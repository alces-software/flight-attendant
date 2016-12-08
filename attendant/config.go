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

var TemplateSets = map[string]string {
  "default": "https://s3-eu-west-1.amazonaws.com/alces-flight/Templates/%s",
}

type Configuration struct {
  AwsRegion string
  AwsAccessKey string
  AwsSecretKey string
  TemplateSet string

  AccessKeyName string
  ApplianceInstanceType string
  MasterInstanceType string
  ComputeInstanceType string
}

var config *Configuration

func Config() *Configuration {
  if config != nil {
    return config
  }
  config = &Configuration{
    AwsRegion: "us-east-1",
    AccessKeyName: "flight-admin",
    ApplianceInstanceType: ApplianceInstanceTypes[0],
  }
  return config
}

func (c *Configuration) IsValidKeyPair() bool {
  return IsValidKeyPairName(c.AccessKeyName)
}
