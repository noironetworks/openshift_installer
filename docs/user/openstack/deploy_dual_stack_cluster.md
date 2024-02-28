# Creating a dual-stack cluster on OpenStack

## Table of Contents

- [Prerequisites](#prerequisites)
- [Creating DualStack Networks for the cluster](#creating-dualstack-networks-for-the-cluster)
- [Creating DualStack API and Ingress VIPs Ports for the cluster](#creating-dualstack-api-and-ingress-vips-for-the-cluster)
- [Deploy OpenShift](#deploy-openshift)

## Prerequisites

* Installation with dual-stack is only allowed when using one OpenStack network with one IPv4 and IPv6 subnet.
* API and Ingress VIPs ports needs to pre-created by the user and the addresses specified in the `install-config.yaml`.
* Add the IPv6 Subnet to a neutron router to provide router advertisements.
* The dualstack network MTU must accomodate the minimun MTU for IPv6, which is 1280, and OVN-Kubernetes encapsulation overhead, which is 100.

Additional prerequisites are listed at the [OpenStack Platform Customization docs](./customization.md)

**Note**: Converting a IPv4 single-stack cluster to a dual-network cluster network is not supported by OpenStack.

## Creating Dual-Stack Networks for the cluster

You must create one network and add one IPv4 subnet and another IPv6 subnet with either slaac, stateless or stateful modes. Also,
you must add the IPv6 subnet to a router. Here is an example:

```sh
$ openstack network create --project <project-name> --share --external --provider-physical-network <physical-network> --provider-network-type flat dualstack
$ openstack subnet create --project <project-name> subnet-v4 --subnet-range 192.168.25.0/24 --dhcp --dns-nameserver <nameserver> --network dualstack
$ openstack subnet create --project <project-name> subnet-v6 --subnet-range fd2e:6f44:5dd8:c956::/64 --dhcp  --network dualstack --ip-version 6 --ipv6-ra-mode slaac --ipv6-address-mode slaac
$ openstack router add subnet <router-id> subnet-v6
```

Note the example above creates a provider network, but a creation of a tenant network is also supported, which must be connected to a router for external connectivity.

## Creating Dual-Stack API and Ingress VIPs Ports for the cluster

You must create the API and Ingress VIPs Ports with the following commands:

```sh
$ openstack port create api --network dualstack
$ openstack port create ingress --network dualstack
```

## Deploy OpenShift

Now that the Networking resources are pre-created you can deploy OpenShift. Here is an example of `install-config.yaml`:

```yaml
apiVersion: v1
baseDomain: mydomain.test
featureSet: TechPreviewNoUpgrade
compute:
- name: worker
  platform:
    openstack:
      type: m1.xlarge
  replicas: 3
controlPlane:
  name: master
  platform:
    openstack:
      type: m1.xlarge
  replicas: 3
metadata:
  name: mycluster
networking:
  machineNetwork:
  - cidr: "192.168.25.0/24"
  - cidr: "fd2e:6f44:5dd8:c956::/64"
  clusterNetwork:
  - cidr: 10.128.0.0/14
    hostPrefix: 23
  - cidr: fd01::/48
    hostPrefix: 64
  serviceNetwork:
  - 172.30.0.0/16
  - fd02::/112
platform:
  openstack:
    ingressVIPs: ['192.168.25.79', 'fd2e:6f44:5dd8:c956:f816:3eff:fef1:1bad']
    apiVIPs: ['192.168.25.199', 'fd2e:6f44:5dd8:c956:f816:3eff:fe78:cf36']
    controlPlanePort:
      fixedIPs:
      - subnet:
          name: subnet-v4
      - subnet:
          name: subnet-v6
      network:
        name: dualstack

```
There are important things to note:

The subnets under `platform.openstack.controlPlanePort.fixedIPs` can contain both id or name. The same applies to the network `platform.openstack.controlPlanePort.network`. Dual-stack clusters are only supported with `featureSet: TechPreviewNoUpgrade`