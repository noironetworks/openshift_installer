openshift-create-cluster
===============

Role to deploy openshift cluster

Requirements
------------

Overcloud must be deployed, and openshift-prereq playbook executed. Additionally images must exist.
Meta folder contain an import of a role to create the authentication in Openstack




Example Playbook
----------------

````
- name: Create needed Resources in Openstack and boot a bastion host
  hosts: controllernode
  become: False
  gather_facts: no
  vars:
    ansible_python_interpreter: /bin/python3
  roles:
    - role: openshift-create-cluster
      action: provision_bastion_host

- name: Configure Bastion and start Boostrapping of the OCP Cluster
  hosts: "bastion"
  become: False
  gather_facts: no
  roles:
    - role: openshift-create-cluster
      action: configure_server

````
Execution
-------
````
[ansible_deployer@si0vmc2903 ~/repositories/ocp]$  ansible-playbook  -i ../ocp-deA-inventory/ -i ../ocp-inventory  -i ../dev-env-inventory/ -i ../ocp-inventory-creds/ type__openshift_installation.yaml --vault-password-file ../vault_password_file

````

License
-------

TBD

Author Information
------------------

Felipe Goikoetxea felipe.goikoetxea@bshg.com Sean O' Gorman sogorman@redhat.com
redacted SDDC Team


