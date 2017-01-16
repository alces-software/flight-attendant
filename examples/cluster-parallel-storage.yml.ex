ClusterName: '%CLUSTER_NAME%'
AccessKeyName: '%ACCESS_KEY_NAME%'
AccessUsername: '%ACCESS_USERNAME%'
OSSInstanceType: 'c3.8xlarge-650GB-10g'
MDSInstanceType: 'c3.large-32GB-mod'
OSSGroupSize: 2
FlightVPC: '%VPC%'
Domain: '%DOMAIN%'
NetworkingPool: '%NETWORK_POOL%'
NetworkingIndex: '%NETWORK_INDEX%'
PrvSubnet: '%PRIVATE_SUBNET%'
MgtSubnet: '%MANAGEMENT_SUBNET%'
PlacementGroup: '%PLACEMENT_GROUP%'
MasterPrivateIP: '%MASTER_IP%'
FlightFeatures: ''
FlightProfileBucket: ''
FlightProfiles: ''
# Fill these in appropriately for your existing cluster. Refer to
# `alces about identity` on your master node.
#
# Note: you should also have the `configure-beegfs` feature enabled on
# members of your cluster.  Either by supplying it in the initial
# launch configuration file, or via `alces customize apply
# configure-beegfs`.
ClusterUUID: ''
ClusterSecurityToken: ''
