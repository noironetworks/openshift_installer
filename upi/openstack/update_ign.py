import base64
import json
import os
import shutil
import tarfile
import yaml


#The script does the following things:
#Update boostrap.ign with hostname and CA certs and with additional network-scripts.
#According to the number of the master count, create the JSON files, and add hostname/network-scripts.
#According to the number of the worker count, create the JSON files, and add hostname/network-scripts.

# Read inventory.yaml for CiscoACI CNI variable
original_inventory = processed_inventory = "inventory.yaml"
with open(original_inventory, 'r') as stream:
    try:
        localhost = yaml.safe_load(stream)['all']['hosts']['localhost']
        inventory = localhost['aci_cni']
    except yaml.YAMLError as exc:
        print(exc)

# Get accprovision tar path from inventory
try:
    acc_provision_tar = inventory['acc_provision_tar']
    os_subnet_range = localhost['os_subnet_range']
except:
    print("inventory.yaml should have acc_provision_tar and os_subnet_range fields")

# Read acc-provision for vlan values
extract_to = './accProvisionTar'
tar = tarfile.open(acc_provision_tar, "r:gz")
tar.extractall(extract_to)
tar.close()

data = ''
for filename in os.listdir(extract_to):
    if 'ConfigMap-aci-containers-config' in filename:
        filepath = "%s/%s" % (extract_to, filename)
        with open(filepath, 'r') as stream:
            try:
                data = yaml.safe_load(stream)['data']['host-agent-config']
            except yaml.YAMLError as exc:
                print(exc)

# Extract host-agent-config and obtain vlan values
try:
    json_data = json.loads(data)
    aci_infra_vlan = json_data['aci-infra-vlan']
    service_vlan = json_data['service-vlan']
    app_profile = json_data['app-profile']
except:
    print("Couldn't extract host-agent-config from aci-containers ConfigMap")

# Delete acc_provisionTar that was untarred previously
try:
    shutil.rmtree(extract_to)
except OSError as e:
    print ("Error: %s - %s." % (e.filename, e.strerror))

if 'mtu' not in inventory['network_interfaces']['opflex']:
    neutron_network_mtu = "1500"
else:
    neutron_network_mtu = str(inventory['network_interfaces']['opflex']['mtu'])

# Set infra_vlan field in inventory.yaml using accprovision tar value
try:
    with open(original_inventory, 'r') as stream:
        cur_yaml = yaml.safe_load(stream)
        cur_yaml['all']['hosts']['localhost']['aci_cni']['app_profile'] = app_profile
        cur_yaml['all']['hosts']['localhost']['aci_cni']['infra_vlan'] = aci_infra_vlan
        cur_yaml['all']['hosts']['localhost']['aci_cni']['service_vlan'] = service_vlan
        cur_yaml['all']['hosts']['localhost']['aci_cni']['network_interfaces']['opflex']['mtu'] = neutron_network_mtu

    if cur_yaml:
        with open(processed_inventory,'w') as yamlfile:
           yaml.safe_dump(cur_yaml, yamlfile)
except:
    print("Unable to edit inventory.yaml")
try:
    node_interface = inventory['network_interfaces']['node']['name']
    opflex_interface = inventory['network_interfaces']['opflex']['name']
    master_count = localhost['os_cp_nodes_number']
    worker_count = localhost['os_compute_nodes_number']
except:
    print("Relevant Fields are missing from inventory.yaml ")

infra_vlan = str(aci_infra_vlan)
infra_id = os.environ.get('INFRA_ID', 'openshift').encode()

def update(hostname,ignition):
    files = ignition['storage'].get('files', [])
    if 'bootstrap' in hostname.decode():
        ca_cert_path = os.environ.get('OS_CACERT', '')
        if ca_cert_path:
            with open(ca_cert_path, 'r') as f:
                ca_cert = f.read().encode()
                ca_cert_b64 = base64.standard_b64encode(ca_cert).decode().strip()

            files.append(
                {
                    'path': '/opt/openshift/tls/cloud-ca-cert.pem',
                    'mode': 420,
                    'contents': {
                        'source': 'data:text/plain;charset=utf-8;base64,' + ca_cert_b64,
                        'verification': {}
                    },
                    'filesystem': 'root',
                })

    hostname_b64 = base64.standard_b64encode(hostname).decode().strip()
    files.append(
        {
            'path': '/etc/hostname',
            'mode': 420,
            'contents': {
                'source': 'data:text/plain;charset=utf-8;base64,' + hostname_b64,
                'verification': {}
            },
            'filesystem': 'root',
        })
    ifcfg_ens3 = ("""TYPE=Ethernet
    DEVICE=""" + node_interface + """
    ONBOOT=yes
    BOOTPROTO=dhcp
    DEFROUTE=yes
    PROXY_METHOD=none
    BROWSER_ONLY=no
    MTU=""" + neutron_network_mtu + """
    IPV4_FAILURE_FATAL=no
    IPV6INIT=no""").encode()

    ifcfg_ens3_b64 = base64.standard_b64encode(ifcfg_ens3).decode().strip()

    files.append(
        {
            'path': '/etc/sysconfig/network-scripts/ifcfg-ens3',
            'mode': 420,
            'contents': {
                'source': 'data:text/plain;charset=utf-8;base64,' + ifcfg_ens3_b64,
                'verification': {}
            },
            'filesystem': 'root',
        })

    ifcfg_ens4 = ("""TYPE=Ethernet
    DEVICE=""" + opflex_interface + """
    ONBOOT=yes
    BOOTPROTO=dhcp
    DEFROUTE=no
    PROXY_METHOD=none
    BROWSER_ONLY=no
    MTU=""" + neutron_network_mtu + """
    IPV4_FAILURE_FATAL=no
    IPV6INIT=no""").encode()

    ifcfg_ens4_b64 = base64.standard_b64encode(ifcfg_ens4).decode().strip()

    files.append(
        {
            'path': '/etc/sysconfig/network-scripts/ifcfg-ens4',
            'mode': 420,
            'contents': {
                'source': 'data:text/plain;charset=utf-8;base64,' + ifcfg_ens4_b64,
                'verification': {}
            },
            'filesystem': 'root',
        })

    opflex_conn = ("""VLAN=yes
    TYPE=Vlan
    PHYSDEV=""" + opflex_interface + """
    VLAN_ID=""" + infra_vlan + """
    REORDER_HDR=yes
    GVRP=no
    MVRP=no
    PROXY_METHOD=none
    BROWSER_ONLY=no
    BOOTPROTO=dhcp
    DEFROUTE=no
    IPV4_FAILURE_FATAL=no
    IPV6INIT=no
    NAME=opflex-conn
    DEVICE=""" + opflex_interface + """.""" + infra_vlan + """
    ONBOOT=yes
    MTU=""" + neutron_network_mtu).encode()

    opflex_conn_b64 = base64.standard_b64encode(opflex_conn).decode().strip()

    files.append(
        {
            'path': '/etc/sysconfig/network-scripts/ifcfg-opflex-conn',
            'mode': 420,
            'contents': {
                'source': 'data:text/plain;charset=utf-8;base64,' + opflex_conn_b64,
                'verification': {}
            },
            'filesystem': 'root',
        })

    route_opflex_conn = """ADDRESS0=224.0.0.0
    NETMASK0=240.0.0.0
    METRIC0=1000""".encode()

    route_opflex_conn_b64 = base64.standard_b64encode(route_opflex_conn).decode().strip()

    files.append(
        {
            'path': '/etc/sysconfig/network-scripts/route-opflex-conn',
            'mode': 420,
            'contents': {
                'source': 'data:text/plain;charset=utf-8;base64,' + route_opflex_conn_b64,
                'verification': {}
            },
            'filesystem': 'root',
        })

    ignition['storage']['files'] = files
    return ignition


with open('bootstrap.ign', 'r') as f:
    ignition = json.load(f)
bootstrap_hostname = infra_id + b'-bootstrap\n'
ignition = update(bootstrap_hostname,ignition)
with open('bootstrap.ign', 'w') as f:
    json.dump(ignition, f)

for index in range(0,master_count):
    master_hostname = infra_id + b'-master-' + str(index).encode() + b'\n'
    with open('master.ign', 'r') as f:
        ignition = json.load(f)
    ignition = update(master_hostname,ignition)
    with open(infra_id.decode() + '-master-' + str(index) + '-ignition.json', 'w') as f:
        json.dump(ignition, f)

for index in range(0,worker_count):
    master_hostname = infra_id + b'-worker-' + str(index).encode() + b'\n'
    with open('worker.ign', 'r') as f:
        ignition = json.load(f)
    ignition = update(master_hostname,ignition)
    with open(infra_id.decode() + '-worker-' + str(index) + '-ignition.json', 'w') as f:
        json.dump(ignition, f)
