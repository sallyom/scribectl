# Multiple Cluster/Multiple Kubeconfig Contexts Rsync Database Example

This example will sync data from mysql database persistent volumes
For this example, sync will happen between 2 clusters. Data will be synced
from cluster-name `api-test-example-com:6443` to cluster-name `kind-kind`

***https://github.com/backube/scribe checked out at ../scribe***

### Build Scribe
$ make scribe
$ mv scribe /usr/local/bin
### Merge Kubeconfigs (if clusters not already in a single kubeconfig)

~/kubeconfig1 with context `kind-kind` and cluster-name `kind-kind`
~/kubeconfig2 with context `admin` and cluster-name `api-test-example-com:6443`
```console
$ export KUBECONFIG=~/kubeconfig1:~/kubeconfig2

You can view config with the following commands:
$ kubectl config view
$ kubectl config get-clusters
$ kubectl config get-contexts
```

### Create source application with:

```console
$ kubectl --context admin create ns source
$ kubectl --context admin -n source apply -f ../scribe/examples/source-database/
```

### Create a database or other changes in the mysql database:

```console
$ kubectl --context admin exec --stdin --tty -n source `kubectl --context admin get pods -n source | grep mysql | awk '{print $1}'` -- /bin/bash
$ mysql -u root -p$MYSQL_ROOT_PASSWORD
> create database my_new_database;
> exit
$ exit
```

### Create a replication destination:

```console
$ kubectl --context kind-kind create ns dest
$ scribe new-destination \
     --dest-namespace dest \
     --dest-access-mode ReadWriteOnce \
     --dest-copy-method Snapshot \
     --dest-kube-context kind-kind \
     --dest-kube-clustername kind-kind
I0302 09:28:35.028745 4174293 options.go:248] ReplicationDestination dest-scribe-destination created in namespace dest
```
Save the rsync address from the destination to pass to the new-source:
```console
$ address=$(kubectl --context kind-kind get replicationdestination/dest-scribe-destination  -n dest --template={{.status.rsync.address}})
```
### Now, create replication source:

```console
$ scribe new-source \
     --address ${address} \
     --source-namespace source \
     --source-copy-method Snapshot \
     --source-pvc mysql-pv-claim \
     --source-kube-context admin \
     --source-kube-clustername api-test-example-com:6443
I0302 09:45:19.026520 4181483 options.go:305] ReplicationSource source-scribe-source created in namespace source
```

TODO: add this to scribe CLI
### Obtain an SSH secret from the destination namespace 

```console
$ kubectl --context kind-kind get secret -n dest scribe-rsync-dest-src-dest-scribe-destination -o yaml > /tmp/secret.yaml
$ vi /tmp/secret.yaml
# ^^^ change the namespace to "source"
# ^^^ remove the owner reference (.metadata.ownerReferences)
$ kubectl --context admin apply -f /tmp/secret.yaml
```

TODO: add this to scribe CLI
### Finally, create a database to sync in the destination ns

Find the latest image from the ReplicationDestination, then
use this image to create the PVC

```console
$ kubectl --context kind-kind get replicationdestination dest-scribe-destination -n dest --template={{.status.latestImage.name}}
$ sed -i 's/snapshotToReplace/scribe-dest-database-destination-20201203174504/g' ../scribe/examples/destination-database/mysql-pvc.yaml
$ kubectl --context kind-kind apply -n dest -f ../scribe/examples/destination-database/
```
