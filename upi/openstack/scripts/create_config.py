# The script uses acc-provision generated configmap to write a yaml with ACI fields like app_profile, VLAN values

import base64
import json
import os
import shutil
import tarfile
import yaml
from jinja2 import Environment, FileSystemLoader
from collections import defaultdict

acc_provision_tar = os.environ.get('ACC_PROVISION_TAR').encode()
aci_cni_fields_path = os.environ.get('ACC_FIELDS_PATH').encode()

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
host_agent_data = json.loads(data)
aci_infra_vlan = host_agent_data['aci-infra-vlan']
service_vlan = host_agent_data['service-vlan']
app_profile = host_agent_data['app-profile']
controller_data = json.loads(data_controller)
aci_vrf_dn = controller_data['aci-vrf-dn']
aci_nodebd_dn = controller_data['aci-nodebd-dn']

# Delete acc_provisionTar that was untarred previously
shutil.rmtree(extract_to)

cur_yaml = dict()
cur_yaml['aci_cni'] = {}
cur_yaml['aci_cni']['app_profile'] = app_profile
cur_yaml['aci_cni']['infra_vlan'] = aci_infra_vlan
cur_yaml['aci_cni']['service_vlan'] = service_vlan
cur_yaml['aci_cni']['network_interfaces'] = {}
cur_yaml['aci_cni']['network_interfaces']['node'] = {}
cur_yaml['aci_cni']['network_interfaces']['node']['vrf'] = aci_vrf_dn
cur_yaml['aci_cni']['network_interfaces']['node']['bd'] = aci_nodebd_dn

if cur_yaml:
    with open(aci_cni_fields_path,'w') as yamlfile:
        yaml.safe_dump(cur_yaml, yamlfile)
