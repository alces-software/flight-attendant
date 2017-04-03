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

var Version = "0.6.0-dev"
var FlightRelease = "2017.2-dev"
var ReleaseDate = "Unknown"

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
  "p2.xlarge-1GPU-4C-61GB",
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
  "t2.xlarge-4C-16GB",
  "t2.2xlarge-8C-32GB",
  "x1.16xlarge-64C-076GB",
  "x1.32xlarge-128C-1952GB",
  "r4.large-2C-15.25GB",
  "r4.xlarge-4C-30.5GB",
  "r4.2xlarge-8C-61GB",
  "r4.4xlarge-16C-122GB",
  "r4.8xlarge-32C-244GB",
  "r4.16xlarge-64C-488GB",
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

var AwsRegions = []string {
  "ap-northeast-1",
  "ap-northeast-2",
  "ap-south-1",
  "ap-southeast-1",
  "ap-southeast-2",
  "ca-central-1",
  "eu-west-1",
  "eu-west-2",
  "eu-central-1",
  "sa-east-1",
  "sa-east-1",
  "us-east-1",
  "us-east-2",
  "us-west-1",
  "us-west-2",
}

var DomainParameters = map[string]string {
  "PeerVPC": "%PEER_VPC%",
  "PeerVPCRouteTable": "%PEER_VPC_ROUTE_TABLE%",
  "PeerVPCCIDRBlock": "%PEER_VPC_CIDR_BLOCK%",
  "AllowInternetAccess": "%ALLOW_INTERNET_ACCESS%",
  "VPNCustomerGateway": "%VPN_CUSTOMER_GATEWAY%",
  "DomainNetworkPrefix": "%DOMAIN_NETWORK_PREFIX%",
}

var ClusterComputeParameters = map[string]string {
  "ClusterName": "%CLUSTER_NAME%",
  "AccessKeyName": "%ACCESS_KEY_NAME%",
  "AccessUsername": "%ADMIN_USER_NAME%",

  "SchedulerType": "%SCHEDULER_TYPE%",
  "FlightFeatures": "%COMPUTE_FEATURES%",
  "FlightProfileBucket": "%PROFILE_BUCKET%",
  "FlightProfiles": "%COMPUTE_PROFILES%",

  "ComputeInstanceType": "%COMPUTE_INSTANCE_TYPE%",
  "ComputeInstanceTypeOther": "%COMPUTE_INSTANCE_OVERRIDE%",
  "ComputeSpotPrice": "%COMPUTE_SPOT_PRICE%",
  "AutoscalingPolicy": "%COMPUTE_AUTOSCALING_POLICY%",
  "ComputeInitialNodes": "%COMPUTE_INITIAL_NODES%",
  "AutoscalingGroupLabel": "%COMPUTE_GROUP_LABEL%",

  "ComputeSystemVolumeType": "%COMPUTE_SYSTEM_VOLUME_TYPE%",

  "Domain": "%DOMAIN%",
  "FlightVPC": "%VPC%",
  "MgtSubnet": "%MGT_SUBNET%",
  "PlacementGroup": "%PLACEMENT_GROUP%",
  "NetworkingPool": "%NETWORK_POOL%",
  "NetworkingIndex": "%NETWORK_INDEX%",
  "MasterPrivateIP": "%MASTER_IP%",
  "PrvSubnet": "%PRV_SUBNET%",

  "ClusterUUID": "%CLUSTER_UUID%",
  "ClusterSecurityToken": "%CLUSTER_SECURITY_TOKEN%",

  "ScratchConfiguration": "%SCRATCH_CONFIGURATION%",
  "SwapConfiguration": "%SWAP_CONFIGURATION%",
  "SwapSize": "%SWAP_SIZE%",
  "SwapSizeMax": "%SWAP_SIZE_MAX%",
}

var ClusterNetworkParameters = map[string]string {
  "FlightVPC": "%VPC%",
  "Domain": "%DOMAIN%",
  "NetworkingPool": "%NETWORK_POOL%",
  "NetworkingIndex": "%NETWORK_INDEX%",
  "PubRouteTable": "%PUB_ROUTE_TABLE%",
}

var ClusterMasterParameters = map[string]string {
  "ClusterName": "%CLUSTER_NAME%",
  "AccessKeyName": "%ACCESS_KEY_NAME%",
  "AccessUsername": "%ADMIN_USER_NAME%",
  "AccessNetwork": "%ACCESS_NETWORK%",

  "SchedulerType": "%SCHEDULER_TYPE%",
  "PreloadSoftware": "%PRELOAD_SOFTWARE%",
  "FlightFeatures": "%MASTER_FEATURES%",
  "FlightProfileBucket": "%PROFILE_BUCKET%",
  "FlightProfiles": "%MASTER_PROFILES%",

  "MasterInstanceType": "%MASTER_INSTANCE_TYPE%",
  "MasterInstanceTypeOther": "%MASTER_INSTANCE_OVERRIDE%",

  "VolumeLayout": "%MASTER_VOLUME_LAYOUT%",
  "VolumeEncryptionPolicy": "%MASTER_VOLUME_ENCRYPTION_POLICY%",
  "MasterSystemVolumeSize": "%MASTER_SYSTEM_VOLUME_SIZE%",
  "MasterSystemVolumeType": "%MASTER_SYSTEM_VOLUME_TYPE%",
  "HomeVolumeSize": "%MASTER_APPS_VOLUME_SIZE%",
  "AppsVolumeSize": "%MASTER_APPS_VOLUME_SIZE%",
  "HomeVolumeType": "%MASTER_HOME_VOLUME_TYPE%",
  "AppsVolumeType": "%MASTER_APPS_VOLUME_TYPE%",

  "Domain": "%DOMAIN%",
  "FlightVPC": "%VPC%",
  "PubSubnet": "%PUB_SUBNET%",
  "MgtSubnet": "%MGT_SUBNET%",
  "PlacementGroup": "%PLACEMENT_GROUP%",
  "NetworkingPool": "%NETWORK_POOL%",
  "NetworkingIndex": "%NETWORK_INDEX%",
  "PrvSubnet": "%PRV_SUBNET%",
  "AllowInternetAccess": "%ALLOW_INTERNET_ACCESS%",

  "ScratchConfiguration": "%SCRATCH_CONFIGURATION%",
  "SwapConfiguration": "%SWAP_CONFIGURATION%",
  "SwapSize": "%SWAP_SIZE%",
  "SwapSizeMax": "%SWAP_SIZE_MAX%",
}

var DomainApplianceParameters = map[string]string {
  "AccessKeyName": "%ACCESS_KEY_NAME%",
  "AccessNetwork": "%ACCESS_NETWORK%",

  "FlightProfileBucket": "%PROFILE_BUCKET%",
  "FlightProfiles": "%APPLIANCE_PROFILES%",

  "ApplianceInstanceType": "%APPLIANCE_INSTANCE_TYPE%",

  "Domain": "%DOMAIN%",
  "FlightVPC": "%VPC%",
  "PubSubnet": "%PUB_SUBNET%",
  "MgtSubnet": "%MGT_SUBNET%",
  "AllowInternetAccess": "%ALLOW_INTERNET_ACCESS%",
}

var SiloParameters = map[string]string {
  "AccessKeyName": "%ACCESS_KEY_NAME%",

  "FlightProfileBucket": "%PROFILE_BUCKET%",
  "FlightProfiles": "%APPLIANCE_PROFILES%",
  "FlightFeatures": "",

  "OSSInstanceType": "%OSS_INSTANCE_TYPE%",
  "OSSGroupSize": "%OSS_GROUP_SIZE%",
  "MDSInstanceType": "%MDS_INSTANCE_TYPE%",

  "Domain": "%DOMAIN%",
  "FlightVPC": "%VPC%",
  "PrvSubnet": "%PRV_SUBNET%",
  "MgtSubnet": "%MGT_SUBNET%",
  "PlacementGroup": "%PLACEMENT_GROUP%",
  "MasterPrivateIP": "%MASTER_IP%",

  // XXX
  "ClusterSecurityToken": "39f05372-f870-11e6-9f50-7831c1c0e63c",
  "ClusterUUID": "39f05372-f870-11e6-9f50-7831c1c0e63c",
}

var BasicApplianceParameters = map[string]string {
  "AccessKeyName": "%ACCESS_KEY_NAME%",
  "AccessUsername": "%ADMIN_USER_NAME%",
  "AccessNetwork": "%ACCESS_NETWORK%",

  "FlightFeatures": "%APPLIANCE_FEATURES%",
  "FlightProfileBucket": "%PROFILE_BUCKET%",
  "FlightProfiles": "%APPLIANCE_PROFILES%",

  "ApplianceInstanceType": "%APPLIANCE_INSTANCE_TYPE%",

  "Domain": "%DOMAIN%",
  "FlightVPC": "%VPC%",
  "PubSubnet": "%PUB_SUBNET%",
  "AllowInternetAccess": "%ALLOW_INTERNET_ACCESS%",
}

var SoloParameters = map[string]string {
  "ClusterName": "%CLUSTER_NAME%",
  "AccessKeyName": "%ACCESS_KEY_NAME%",
  "AccessUsername": "%ADMIN_USER_NAME%",
  "AccessNetwork": "%ACCESS_NETWORK%",

  "SchedulerType": "%SCHEDULER_TYPE%",
  "PreloadSoftware": "%PRELOAD_SOFTWARE%",
  "FlightFeatures": "%MASTER_FEATURES%",
  "FlightProfileBucket": "%PROFILE_BUCKET%",
  "FlightProfiles": "%MASTER_PROFILES%",

  "MasterInstanceType": "%MASTER_INSTANCE_TYPE%",
  "MasterInstanceTypeOther": "%MASTER_INSTANCE_OVERRIDE%",

  "ComputeInstanceType": "%COMPUTE_INSTANCE_TYPE%",
  "ComputeInstanceTypeOther": "%COMPUTE_INSTANCE_OVERRIDE%",
  "ComputeSpotPrice": "%COMPUTE_SPOT_PRICE%",
  "AutoscalingPolicy": "%COMPUTE_AUTOSCALING_POLICY%",
  "ComputeInitialNodes": "%COMPUTE_INITIAL_NODES%",

  "VolumeLayout": "%MASTER_VOLUME_LAYOUT%",
  "VolumeEncryptionPolicy": "%MASTER_VOLUME_ENCRYPTION_POLICY%",
  "MasterSystemVolumeSize": "%MASTER_SYSTEM_VOLUME_SIZE%",
  "MasterSystemVolumeType": "%MASTER_SYSTEM_VOLUME_TYPE%",
  "HomeVolumeSize": "%MASTER_APPS_VOLUME_SIZE%",
  "AppsVolumeSize": "%MASTER_APPS_VOLUME_SIZE%",
  "HomeVolumeType": "%MASTER_HOME_VOLUME_TYPE%",
  "AppsVolumeType": "%MASTER_APPS_VOLUME_TYPE%",
  "ComputeSystemVolumeType": "%COMPUTE_SYSTEM_VOLUME_TYPE%",
  "ScratchConfiguration": "%SCRATCH_CONFIGURATION%",
  "SwapConfiguration": "%SWAP_CONFIGURATION%",
  "SwapSize": "%SWAP_SIZE%",
  "SwapSizeMax": "%SWAP_SIZE_MAX%",
}

var ParameterSets = map[string]*map[string]string {
  "domain": &DomainParameters,
  "cluster-network": &ClusterNetworkParameters,
  "cluster-master": &ClusterMasterParameters,
  "cluster-compute": &ClusterComputeParameters,
  "solo": &SoloParameters,
  "directory": &DomainApplianceParameters,
  "monitor": &DomainApplianceParameters,
  "storage-manager": &BasicApplianceParameters,
  "access-manager": &BasicApplianceParameters,
}
