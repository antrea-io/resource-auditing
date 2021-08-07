
# resource-auditing

## Overview
This project provides an auditing system for network policy-related, Kubernetes
resources and Antrea specific CRDs that tracks creations, updates, and deletions
of these resources, stored as YAML files backed by a Git repository. The system
also comes with a CLI for querying and filtering the repository for changes 
based onfilters like date range or service account responsible for those 
changes, as well as a tagging and rollback feature for reverting the cluster 
state if the current cluster state is undesirable. A webUI service is linked 
to the repository, allowing for easy visualization of the entire history of
resource configurations.

## Getting Started
Ensure `kubectl` is running correctly prior to getting started. A label is used
to specify the node to run all audit services on. The `nodeAffinity` field is
used to schedule the Pods to the correct Node(s) and uses Node labels to
determine them. The label can be applied with:
```bash
kubectl label nodes <node-name> audit=target
```
Connect to the control Node and copy `audit-policy.yaml` and `audit-config.yaml`
to `/etc/kubernetes/manifests`. Run the following commands to create the
directory the repository will be stored on.
```bash
mkdir -p /data/antrea-audit
```
Modify the kube-apiserver.yaml manifest by adding the following lines to the
manifest:
```yaml
    - command
    - --kube-apiserver
    - --audit-policy-file=/path/to/audit-policy.yaml
    - - --audit-config-file=/path/to/audit-config.yaml
...
    volumeMounts:
    - mountPath: /path/to/audit-policy.yaml
      name: audit-policy
      readOnly: true
    - mountPath: /path/to/audit-config.yaml
      name: audit-config
      readOnly: true
...
  volumes:
  - hostPath:
      path: /path/to/audit-policy.yaml
    name: audit-policy
  - hostPath:
      path: /path/to/audit-config.yaml
    name: audit-config
```
Exit the control Node. To deploy the most recent version of resource-auditing,
use the checked in [deployment yaml](https://github.com/antrea-io/resource-auditing/tree/main/reference-manifests/audit-webhook.yaml):
```bash
kubectl apply -f https://raw.githubusercontent.com/antrea-io/resource-auditing/tree/main/build/yamls
```

## Contributing
The Antrea community welcomes new contributors. We are waiting for your PRs!
* This project follows the [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/master/code-of-conduct.md).
* Check out [Open Issues](https://github.com/antrea-io/resource-auditing/issues)
