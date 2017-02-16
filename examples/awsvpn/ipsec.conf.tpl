config setup
	charondebug="cfg 2, ike 3"

conn %default
	leftauth=psk
	rightauth=psk
	ike=aes256-sha256-modp2048s256,aes128-sha1-modp1024!
	ikelifetime=28800s
	aggressive=no
	esp=aes128-sha256-modp2048s256,aes128-sha1-modp1024!
	lifetime=3600s
	type=tunnel
	dpddelay=10s
	dpdtimeout=30s
	keyexchange=ikev1
	rekey=yes
	reauth=no
	dpdaction=restart
	closeaction=restart
	left=%defaultroute
	leftsubnet=0.0.0.0/0,::/0
	rightsubnet=0.0.0.0/0,::/0
	leftupdown=/etc/strongswan.d/ipsec-vti.sh
	installpolicy=yes
	compress=no
	mobike=no

conn AWS-VPC-GW1
	left=10.3.1.10
	right=_OUTSIDE_GW1_
	auto=start
	mark=100

conn AWS-VPC-GW2
	left=10.3.1.10
	right=_OUTSIDE_GW2_
	auto=start
	mark=200
