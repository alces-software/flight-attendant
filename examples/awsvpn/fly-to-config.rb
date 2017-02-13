#!/usr/bin/ruby
require 'yaml'
yaml = YAML.load_file(ARGV[0])
puts <<EOF
outside_gws=(#{yaml['tunnel1']['OutsideAwsAddr']} #{yaml['tunnel2']['OutsideAwsAddr']})
inside_customer_gws=(#{yaml['tunnel1']['InsideClientAddr']}/#{yaml['InsideCIDR']} #{yaml['tunnel2']['InsideClientAddr']}/#{yaml['InsideCIDR']})
inside_virtual_prv_gws=(#{yaml['tunnel1']['InsideAwsAddr']}/#{yaml['InsideCIDR']} #{yaml['tunnel2']['InsideAwsAddr']}/#{yaml['InsideCIDR']})
psks=(#{yaml['tunnel1']['SharedKey']} #{yaml['tunnel2']['SharedKey']})
customer_gateway_asn=#{yaml['ClientASN']}
virtual_prv_gw_asns=(#{yaml['tunnel1']['AwsASN']} #{yaml['tunnel2']['AwsASN']})
neighbor_ip_addrs=(#{yaml['tunnel1']['InsideAwsAddr']} #{yaml['tunnel2']['InsideAwsAddr']})
EOF
