hostname _HOSTNAME_
password _PASSWORD_
enable password _ENABLE_PASSWORD_
!
route-map RM_SET_SRC permit 10
  set src _LOCAL_IP_
ip protocol bgp route-map RM_SET_SRC
!
line vty
