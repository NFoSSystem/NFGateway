#!/bin/bash

cd eth
go build eth2ip.go
cd ..
cd ip
go build ip2ip.go
cd ..
scp eth/eth2ip abanfi8@c220g2-011006.wisc.cloudlab.us:/users/abanfi8/.
scp ip/ip2ip abanfi8@c220g2-011006.wisc.cloudlab.us:/users/abanfi8/.
