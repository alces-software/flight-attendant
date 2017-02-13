#!/bin/bash
set -e

. local-config.rc
. vpn-config.rc

mkdir -p output/etc/strongswan.d
mkdir -p output/etc/quagga

sed \
    -e "s/_HOSTNAME_/$hostname/g" \
    -e "s/_ENABLE_PASSWORD_/$enable_password/g" \
    -e "s/_PASSWORD_/$password/g" \
    -e "s,_LOCAL_NET_,$local_net,g" \
    -e "s/_NEIGHBOR_IP_ADDR1_/${neighbor_ip_addrs[0]}/g" \
    -e "s/_NEIGHBOR_IP_ADDR2_/${neighbor_ip_addrs[1]}/g" \
    -e "s/_VIRTUAL_PRV_GW_ASN1_/${virtual_prv_gw_asns[0]}/g" \
    -e "s/_VIRTUAL_PRV_GW_ASN2_/${virtual_prv_gw_asns[1]}/g" \
    -e "s/_CUSTOMER_GW_ASN_/${customer_gateway_asn}/g" \
    bgpd.conf.tpl > output/etc/quagga/bgpd.conf
cp daemons.tpl output/etc/quagga/daemons
sed \
    -e "s/_LOCAL_IP_/$local_ip/g" \
    -e "s/_HOSTNAME_/$hostname/g" \
    -e "s/_ENABLE_PASSWORD_/$enable_password/g" \
    -e "s/_PASSWORD_/$password/g" \
    zebra.conf.tpl > output/etc/quagga/zebra.conf

chmod 0640 output/etc/quagga/*
chown quagga:quagga output/etc/quagga/*

sed \
    -e "s/_OUTSIDE_GW1_/${outside_gws[0]}/g" \
    -e "s/_OUTSIDE_GW2_/${outside_gws[1]}/g" \
    ipsec.conf.tpl > output/etc/ipsec.conf
chmod 0644 output/etc/ipsec.conf

sed \
    -e "s/_LOCAL_IP_/$local_ip/g" \
    -e "s/_OUTSIDE_GW1_/${outside_gws[0]}/g" \
    -e "s/_OUTSIDE_GW2_/${outside_gws[1]}/g" \
    -e "s/_PSK_GW1_/${psks[0]}/g" \
    -e "s/_PSK_GW2_/${psks[1]}/g" \
    ipsec.secrets.tpl > output/etc/ipsec.secrets
chmod 0600 output/etc/ipsec.secrets

sed \
    -e "s,_INSIDE_CUSTOMER_GW1_,${inside_customer_gws[0]},g" \
    -e "s,_INSIDE_CUSTOMER_GW2_,${inside_customer_gws[1]},g" \
    -e "s,_INSIDE_VIRTUAL_PRV_GW1_,${inside_virtual_prv_gws[0]},g" \
    -e "s,_INSIDE_VIRTUAL_PRV_GW2_,${inside_virtual_prv_gws[1]},g" \
    ipsec-vti.sh.tpl  > output/etc/strongswan.d/ipsec-vti.sh
chmod 0700 output/etc/strongswan.d/ipsec-vti.sh

cp charon.conf.tpl output/etc/strongswan.d/charon.conf
chmod 0640 output/etc/strongswan.d/charon.conf
