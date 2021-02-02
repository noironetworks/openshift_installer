import yaml;
path = "openshift/99_openshift-cluster-api_worker-machineset-0.yaml"
data = yaml.safe_load(open(path));
networks = data['spec']['template']['spec']['providerSpec']['value'].get('networks', [])
networks.append(
    {
        'filter': {},
        'subnets': [
            {
                'filter': {
                    'name': str(data['metadata']['labels']['machine.openshift.io/cluster-api-cluster'] + '-acicontainers-nodes'),
                    'tags': str('openshiftClusterID=' + data['metadata']['labels']['machine.openshift.io/cluster-api-cluster'])
                }
            }
        ]
    }
)
data['spec']['template']['spec']['providerSpec']['value']['networks'] = networks
open(path, "w").write(yaml.dump(data, default_flow_style=False))
