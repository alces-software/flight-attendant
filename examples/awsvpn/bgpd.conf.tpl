hostname _HOSTNAME_
password _PASSWORD_
enable password _ENABLE_PASSWORD_
!
log file /var/log/quagga/bgpd
!debug bgp events
!debug bgp zebra
!debug bgp updates
!
router bgp _CUSTOMER_GW_ASN_
  network _LOCAL_NET_
  neighbor _NEIGHBOR_IP_ADDR1_ remote-as _VIRTUAL_PRV_GW_ASN1_
  neighbor _NEIGHBOR_IP_ADDR2_ remote-as _VIRTUAL_PRV_GW_ASN2_

  ! Uncomment the line below if you prefer to use 'Connection B' as
  ! your backup (Connection A will be used as your primary for all
  ! traffic). By default if you do not uncomment the next lines,
  ! traffic can be sent and received down both of your connections at
  ! any time (asymmetric routing).
  !neighbor _NEIGHBOR_IP_ADDR2_ route-map RM_LOWER_PRIORITY out
  network 0.0.0.0
!
route-map RM_LOWER_PRIORITY permit 10
  set as-path prepend _CUSTOMER_GW_ASN_ _CUSTOMER_GW_ASN_ _CUSTOMER_GW_ASN_
!
line vty
