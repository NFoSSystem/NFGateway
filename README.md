# NFGateway

NFGateway is the main module of the Network Function over Serverless System (NFoS System).

The NFoS System is the result of the M.S. Thesis in Computer Science entitled "*Modular Extension of a Serverless Platform for the execution of Virtual Network Functions*" authored by Alessandro Banfi (*a.banfi8 __at-character__ campus.unimib.it*).

The following picture shows the NFoS System with a series of components with the NFGateway module as well.

![NFoS System](https://github.com/NFoSSystem/NFGateway/blob/develop/nfos-arch.png?raw=true)

Despite the system has been designed to rely on Apache Openwhisk, with few changes other Serverless platforms, both open source and vendor specific on public cloud, can be made suitable to host a set of Serverless Network Functions.

The NFGateway perform the following actions:
* receives messages coming from the outside of the system and using protocols different by HTTP/S (e.g. TCP, UDP and others);
* chooses between the set of available and registered Serverless Network Functions the target one;
* forwards the message to an instance of the Serverless NF previously selected.

## Requirements

Here the requirements necessary to set up the NFoS System:
* [Apache Openwhisk](https://github.com/apache/openwhisk.git) version 1.0.0
* [Redis database](https://github.com/redis/redis.git)
* [wsk](https://github.com/apache/openwhisk-cli.git) - Openwhisk Command Line Interface
* [faasnat](https://github.com/NFoSSystem/faasnat) - Serverless version of the NAT function
* [faasdhcp](https://github.com/NFoSSystem/faasdhcp.git) - Serverless version of the DHCP function

Together with [nflib](https://github.com/NFoSSystem/nflib.git), faasnat and faasdhcp are available with their source code and are belonging to the NFoS System project here on Github.

## Installation

Here the steps to follow:
* clone the NFGateway repository and checkout the develop branch
* build the nfgateway executable with the command `go build`
* start Openwhisk, the Redis server, obtain a REST API token for Openwhisk and get the name of an available ethernet interface (execute the `ifconfig` command on a *nix system)
* execute the following command in a terminal as a root user to start the nfgateway process
```bash
# ~ sudo ./nfgateway eth-interface-name openwhisk-api-address-and-port openwhisk-auth-token redis-address-and-port
```
here you find an example of a command with all the parameters provided using the above syntax
```bash
# ~ sudo ./nfgateway enp6s0f0 172.17.0.1:3233 MjNiYzQ2YjEtNzFmNi00ZWQ1LThjNTQtODE2YWE0ZjhjNTAyOjEyM3pPM3haQ0xyTU42djJCS0sxZFhZRnBYbFBrY2NPRnFtMTJDZEFzTWdSVTRWck5aOWx5R1ZDR3VNREdJd1A 172.17.0.1:6379
```
* in case you would like to deploy the NAT Serverless function follow these steps:
    * clone the [faasnat](https://github.com/NFoSSystem/faasnat.git) repository, checkout the `openwhisk-http` branch
    * modify and execute the script named `build_and_send.sh` to obtain an arhive named *nat-bin.zip*
    * deploy the Serverless function on openwhisk executing the command
```bash
./wsk action create nat nat-bin.zip --main main --docker openwhisk/action-golang-v1.15:nightly
```
* in case you would like to deploy the DHCP Serverless function follow these steps:
    * clone the [faasdhcp](https://github.com/NFoSSystem/faasdhcp.git) repository, checkout the `openwhisk` branch
    * modify and execute the script named `build_and_send.sh` to obtain an arhive named *dhcp-bin.zip*
    * deploy the Serverless function on openwhisk executing the command
```bash
./wsk action create dhcp dhcp-bin.zip --main main --docker openwhisk/action-golang-v1.15:nightly
```