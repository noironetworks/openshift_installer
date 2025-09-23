import yaml

"""
Script does the following things:
- Reads the control plane machine set manifest to extract the OpenShift cluster ID.
- Constructs network and subnet names based on the cluster ID.
- Updates the 'networks' field in the control plane machine set manifest with the constructed network information.
- Iterates over the three master machine manifest files and updates their 'networks' field with the same network information.
"""

# Get the count of worker nodes from install-config.yaml
inventory_file = "inventory.yaml"
with open(inventory_file, 'r') as stream:
    try:
        localhost = yaml.safe_load(stream)['all']['hosts']['localhost']
    except yaml.YAMLError as exc:
        print(exc)
try:
    master_count = localhost['os_cp_nodes_number']
except:
    print("os_cp_nodes_number Field is missing from inventory.yaml")
    exit(1)

# Get the openshift cluster id
machineset_file = (
    f"openshift/99_openshift-machine-api_master-control-plane-machine-set.yaml"
)
with open(machineset_file, "r") as f:
    cpms = yaml.safe_load(f)

if cpms is None:
    print("Unable to read control plane machine set manifest")
    exit(1)

infra_id = cpms["metadata"]["labels"]["machine.openshift.io/cluster-api-cluster"]
if infra_id is None:
    print("Unable to get openshift cluster id from control plane machine set manifest")
    exit(1)

node_network_name = infra_id + "-nodes"
aci_network_name = infra_id + "-acicontainers-nodes"

# Updated networks filter config with node network and aci network names
networks = [
    {"filter": {}, "subnets": [{"filter": {"name": node_network_name}}]},
    {"filter": {}, "subnets": [{"filter": {"name": aci_network_name}}]},
]
cpms["spec"]["template"]["machines_v1beta1_machine_openshift_io"]["spec"][
    "providerSpec"
]["value"]["networks"] = networks

# Update control plane machine set manifest
with open(machineset_file, "w") as f:
    yaml.safe_dump(cpms, f)

# Update master machine manifests
for i in range(master_count):
    master_file = f"openshift/99_openshift-cluster-api_master-machines-{i}.yaml"
    with open(master_file, "r") as f:
        master_machine_manifest = yaml.safe_load(f)
    if master_machine_manifest is None:
        print(f"Unable to read master machine manifest {master_file}")
        exit(1)
    master_machine_manifest["spec"]["providerSpec"]["value"]["networks"] = networks
    with open(master_file, "w") as f:
        yaml.safe_dump(master_machine_manifest, f)
