import base64
import json
import os
import shutil
import tarfile
import yaml
from jinja2 import Environment, FileSystemLoader


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
        with open(filepath, 'r') as stream:
            try:
                data_controller = yaml.safe_load(stream)['data']['controller-config']
            except yaml.YAMLError as exc:
                print(exc)

# Extract host-agent-config and obtain vlan values
try:
    host_agent_data = json.loads(data)
    aci_infra_vlan = host_agent_data['aci-infra-vlan']
    service_vlan = host_agent_data['service-vlan']
    app_profile = host_agent_data['app-profile']

    controller_data = json.loads(data_controller)
    aci_vrf_dn = controller_data['aci-vrf-dn']
    aci_nodebd_dn = controller_data['aci-nodebd-dn']
except:
    print("Couldn't extract host-agent-config from aci-containers ConfigMap")

# Delete acc_provisionTar that was untarred previously
try:
    shutil.rmtree(extract_to)
except OSError as e:
    print ("Error: %s - %s." % (e.filename, e.strerror))

# Set infra_vlan field in inventory.yaml using accprovision tar value
try:
    with open(original_inventory, 'r') as stream:
        cur_yaml = yaml.safe_load(stream)

    cur_yaml['all']['hosts']['localhost']['aci_cni']['app_profile'] = app_profile
    cur_yaml['all']['hosts']['localhost']['aci_cni']['infra_vlan'] = aci_infra_vlan

    if 'node_epg' not in inventory:
        cur_yaml['all']['hosts']['localhost']['aci_cni']['node_epg'] = "aci-containers-nodes"

    cur_yaml['all']['hosts']['localhost']['aci_cni']['service_vlan'] = service_vlan

    if 'network_interfaces' not in inventory:
        cur_yaml['all']['hosts']['localhost']['aci_cni']['network_interfaces'] = dict()

    if 'opflex' not in cur_yaml['all']['hosts']['localhost']['aci_cni']['network_interfaces']:
        cur_yaml['all']['hosts']['localhost']['aci_cni']['network_interfaces']['opflex'] = dict()

    if 'node' not in cur_yaml['all']['hosts']['localhost']['aci_cni']['network_interfaces']:
        cur_yaml['all']['hosts']['localhost']['aci_cni']['network_interfaces']['node'] = dict()

    if 'mtu' not in cur_yaml['all']['hosts']['localhost']['aci_cni']['network_interfaces']['opflex']:
        cur_yaml['all']['hosts']['localhost']['aci_cni']['network_interfaces']['opflex']['mtu'] = 1500

    if 'mtu' not in cur_yaml['all']['hosts']['localhost']['aci_cni']['network_interfaces']['node']:
        cur_yaml['all']['hosts']['localhost']['aci_cni']['network_interfaces']['node']['mtu'] = 1500

    if 'name' not in cur_yaml['all']['hosts']['localhost']['aci_cni']['network_interfaces']['opflex']:
        cur_yaml['all']['hosts']['localhost']['aci_cni']['network_interfaces']['opflex']['name'] = 'ens4'

    if 'name' not in cur_yaml['all']['hosts']['localhost']['aci_cni']['network_interfaces']['node']:
        cur_yaml['all']['hosts']['localhost']['aci_cni']['network_interfaces']['node']['name'] = 'ens3'

    if 'subnet' not in cur_yaml['all']['hosts']['localhost']['aci_cni']['network_interfaces']['opflex']:
        cur_yaml['all']['hosts']['localhost']['aci_cni']['network_interfaces']['opflex']['subnet'] = '192.168.208.0/20'

    cur_yaml['all']['hosts']['localhost']['aci_cni']['network_interfaces']['node']['vrf'] = aci_vrf_dn
    cur_yaml['all']['hosts']['localhost']['aci_cni']['network_interfaces']['node']['bd'] = aci_nodebd_dn

    if cur_yaml:
        with open(processed_inventory,'w') as yamlfile:
           yaml.safe_dump(cur_yaml, yamlfile)
except:
    print("Unable to edit inventory.yaml")
try:
    node_interface = cur_yaml['all']['hosts']['localhost']['aci_cni']['network_interfaces']['node']['name']
    opflex_interface = cur_yaml['all']['hosts']['localhost']['aci_cni']['network_interfaces']['opflex']['name']
    master_count = localhost['os_cp_nodes_number']
    worker_count = localhost['os_compute_nodes_number']
except:
    print("Relevant Fields are missing from inventory.yaml ")

infra_vlan = str(aci_infra_vlan)
infra_id = os.environ.get('INFRA_ID', 'openshift').encode()
neutron_network_mtu = str(cur_yaml['all']['hosts']['localhost']['aci_cni']['network_interfaces']['node']['mtu'])
opflex_network_mtu = str(cur_yaml['all']['hosts']['localhost']['aci_cni']['network_interfaces']['opflex']['mtu'])

def update(hostname,ignition):

    config_data = {}

    ifcfg_ens3 = ("""TYPE=Ethernet
DEVICE=""" + node_interface + """
ONBOOT=yes
BOOTPROTO=dhcp
DEFROUTE=yes
PROXY_METHOD=none
BROWSER_ONLY=no
MTU=""" + neutron_network_mtu + """
IPV4_FAILURE_FATAL=no
IPV6INIT=no
ETHTOOL_OPTS="-K ens3 tx-checksum-ip-generic off"
NAME="System ens3"
UUID=21d47e65-8523-1a06-af22-6f121086f085
""").encode()

    ifcfg_ens3_b64 = base64.standard_b64encode(ifcfg_ens3).decode().strip()

    config_data['ifcfg_ens3'] = {'base64': ifcfg_ens3_b64, 'path': '/etc/sysconfig/network-scripts/ifcfg-ens3'}

    ifcfg_ens4 = ("""TYPE=Ethernet
DEVICE=""" + opflex_interface + """
ONBOOT=yes
BOOTPROTO=dhcp
DEFROUTE=no
PROXY_METHOD=none
BROWSER_ONLY=no
MTU=""" + opflex_network_mtu + """
IPV4_FAILURE_FATAL=no
IPV6INIT=no
ETHTOOL_OPTS="-K ens4 tx-checksum-ip-generic off"
NAME="System ens4"
UUID=e27f182b-d125-2c43-5a30-43524d0229ac
""").encode()

    ifcfg_ens4_b64 = base64.standard_b64encode(ifcfg_ens4).decode().strip()

    config_data['ifcfg_ens4'] = {'base64': ifcfg_ens4_b64, 'path': '/etc/sysconfig/network-scripts/ifcfg-ens4'}

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
MTU=""" + opflex_network_mtu + """
HWADDR=
ETHTOOL_OPTS="-K net0 tx-checksum-ip-generic off"
UUID=eb4377c5-a6d1-f09a-f588-7a6122be32f5
""").encode()

    ifcfg_opflex_conn_b64 = base64.standard_b64encode(opflex_conn).decode().strip()

    config_data['ifcfg_opflex_conn'] = {'base64': ifcfg_opflex_conn_b64, 'path': '/etc/sysconfig/network-scripts/ifcfg-opflex-conn'}

    route_opflex_conn = """ADDRESS0=224.0.0.0
NETMASK0=240.0.0.0
METRIC0=1000
""".encode()

    route_opflex_conn_b64 = base64.standard_b64encode(route_opflex_conn).decode().strip()

    config_data['route_opflex_conn'] = {'base64': route_opflex_conn_b64, 'path': '/etc/sysconfig/network-scripts/route-opflex-conn'}
    if 'storage' not in ignition.keys():
        ignition['storage'] = {}
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

        # Add master and worker network scripts to bootstrap ignition
        env = Environment(loader = FileSystemLoader('./templates'), trim_blocks=True, lstrip_blocks=True)
        template_worker = env.get_template('99_worker-networkscripts.yaml')
        rendered_worker = template_worker.render(config_data)
        worker_b64 = base64.standard_b64encode(rendered_worker.encode()).decode().strip()

        template_master = env.get_template('99_master-networkscripts.yaml')
        rendered_master = template_master.render(config_data)
        master_b64 = base64.standard_b64encode(rendered_master.encode()).decode().strip()

        files.append(
            {
               'path': '/opt/openshift/openshift/99_master-networkscripts.yaml',
               'mode': 420,
               'contents': {
                   'source': 'data:text/plain;charset=utf-8;base64,' + master_b64,
                   'verification': {}
               },
               'filesystem': 'root',
            })

        files.append(
            {
               'path': '/opt/openshift/openshift/99_worker-networkscripts.yaml',
               'mode': 420,
               'contents': {
                   'source': 'data:text/plain;charset=utf-8;base64,' + worker_b64,
                   'verification': {}
               },
               'filesystem': 'root',
            })

    else:
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

         files.append(
             {
                 'path': config_data['ifcfg_ens3']['path'],
                 'mode': 420,
                 'contents': {
                     'source': 'data:text/plain;charset=utf-8;base64,' + config_data['ifcfg_ens3']['base64'],
                     'verification': {}
                 },
                 'filesystem': 'root',
             })

         files.append(
             {
                 'path': config_data['ifcfg_ens4']['path'],
                 'mode': 420,
                 'contents': {
                     'source': 'data:text/plain;charset=utf-8;base64,' + config_data['ifcfg_ens4']['base64'],
                     'verification': {}
                 },
                 'filesystem': 'root',
             })

         files.append(
             {
                 'path': config_data['ifcfg_opflex_conn']['path'],
                 'mode': 420,
                 'contents': {
                     'source': 'data:text/plain;charset=utf-8;base64,' + config_data['ifcfg_opflex_conn']['base64'],
                     'verification': {}
                 },
                 'filesystem': 'root',
             })

         files.append(
             {
                 'path': config_data['route_opflex_conn']['path'],
                 'mode': 420,
                 'contents': {
                     'source': 'data:text/plain;charset=utf-8;base64,' + config_data['route_opflex_conn']['base64'],
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

os.system('cat > ' + infra_id.decode() + '''-bootstrap-ignition.json << EOL
{
  "ignition": {
    "config": {
      "merge": [
        {
          "source": "$(swift stat -v | grep StorageURL | awk -F': ' '{print$2}')/bootstrap/bootstrap.ign"
        }
      ]
    },
    "version": "3.1.0"
  }
}
EOL''')

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
