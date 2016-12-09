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

var InstanceTypes []string = []string{
  "c3.large-2C-3.75GB",
  "c3.xlarge-4C-7.5GB",
  "c3.2xlarge-8C-15GB",
  "c3.4xlarge-16C-30GB",
  "c3.8xlarge-32C-60GB",
  "c4.large-2C-3.75GB",
  "c4.xlarge-4C-7.5GB",
  "c4.2xlarge-8C-15GB",
  "c4.4xlarge-16C-30GB",
  "c4.8xlarge-36C-60GB",
  "d2.xlarge-4C-30.5GB",
  "d2.2xlarge-8C-61GB",
  "d2.4xlarge-16C-122GB",
  "d2.8xlarge-36C-244GB",
  "g2.2xlarge-1GPU-8C-15GB",
  "g2.8xlarge-4GPU-32C-60GB",
  "i2.xlarge-4C-30.5GB",
  "i2.2xlarge-8C-61GB",
  "i2.4xlarge-16C-122GB",
  "i2.8xlarge-32C-244GB",
  "m3.medium-1C-3.75GB",
  "m3.large-2C-7.5GB",
  "m3.xlarge-4C-15GB",
  "m3.2xlarge-8C-30GB",
  "m4.large-2C-8GB",
  "m4.xlarge-4C-16GB",
  "m4.2xlarge-8C-32GB",
  "m4.4xlarge-16C-64GB",
  "m4.10xlarge-40C-160GB",
  "m4.16xlarge-64C-256GB",
  "p2.xlarge-4GPU-4C-61GB",
  "p2.8xlarge-8GPU-32C-488GB",
  "p2.16xlarge-16GPU-64C-732GB",
  "r3.large-2C-15.25GB",
  "r3.xlarge-4C-30.5GB",
  "r3.2xlarge-8C-61GB",
  "r3.4xlarge-16C-122GB",
  "r3.8xlarge-32C-244GB",
  "t2.nano-1C-0.5GB",
  "t2.micro-1C-1GB",
  "t2.small-1C-2GB",
  "t2.medium-2C-4GB",
  "t2.large-2C-8GB",
  "x1.16xlarge-64C-076GB",
  "x1.32xlarge-128C-1952GB",
}

var SystemVolumeTypes = []string {
  "magnetic.standard",
  "general-purpose-ssd.gp2",
}

var OtherVolumeTypes = []string {
  "magnetic.standard",
  "general-purpose-ssd.gp2",
  "throughput-optimized-hdd.st1",
  "cold-hdd.sc1",
}

var SoftwareTypes = []string {
  "-none-",
  "benchmark",
  "bioinformatics",
  "cfd",
  "chemistry",
  "development",
}

var SchedulerTypes = []string {
  "gridscheduler",
  "openlava",
  "pbspro",
  "slurm",
  "torque",
}