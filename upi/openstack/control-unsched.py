import yaml;
path = "manifests/cluster-scheduler-02-config.yml"
data = yaml.safe_load(open(path));
data["spec"]["mastersSchedulable"] = False;
open(path, "w").write(yaml.dump(data, default_flow_style=False))
