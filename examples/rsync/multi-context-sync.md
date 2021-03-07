# Multiple Cluster/Multiple Kubeconfig Contexts Rsync Database Example

This example will sync data from mysql database persistent volumes
For this example, sync will happen between 2 clusters. Data will be synced
from cluster-name `api-test-example-com:6443` to cluster-name `test0123`

***https://github.com/backube/scribe checked out at ../scribe***

### Build Scribe

```bash
$ make scribe
$ mv scribe /usr/local/bin (or add to $PATH)
```

### Merge Kubeconfigs (if clusters not already in a single kubeconfig)

~/kubeconfig1 with context `testuser` and cluster-name `test0123`

~/kubeconfig2 with context `admin` and cluster-name `api-test-example-com:6443`
```bash
$ export KUBECONFIG=~/kubeconfig1:~/kubeconfig2

You can view config with the following commands:
$ kubectl config view
$ kubectl config get-clusters
$ kubectl config get-contexts
```

### Create source application with:

```bash
$ kubectl --context admin create ns source
$ kubectl --context admin -n source apply -f ../scribe/examples/source-database/
```

### Create a database or other changes in the mysql database:

```bash
$ kubectl --context admin exec --stdin --tty -n source `kubectl --context admin get pods -n source | grep mysql | awk '{print $1}'` -- /bin/bash
$ mysql -u root -p$MYSQL_ROOT_PASSWORD
> create database my_new_database;
> exit
$ exit
```

### Create a replication destination:

_If `kubectl get-contexts` shows current context is `testuser` then you can omit the `--dest-kube-context|clustername` flags_

```bash
$ kubectl --context testuser create ns dest
$ scribe new-destination \
     --dest-namespace dest \
     --dest-access-mode ReadWriteOnce \
     --dest-copy-method Snapshot \
     --dest-kube-context testuser \
     --dest-kube-clustername test0123
I0302 09:28:35.028745 4174293 options.go:248] ReplicationDestination dest-scribe-destination created in namespace dest
```
Save the rsync address from the destination to pass to the new-source:
```bash
$ address=$(kubectl --context testuser get replicationdestination/dest-scribe-destination  -n dest --template={{.status.rsync.address}})
$ echo ${address} //to be sure it's not empty, may take a minute to populate
```
### Now, create replication source:

_If `kubectl get-contexts` shows current context is `admin` then can omit the `source-kube-context|clustername` flags_

```bash
$ scribe new-source \
     --address ${address} \
     --source-namespace source \
     --source-copy-method Snapshot \
     --source-pvc mysql-pv-claim \
     --source-kube-context admin \
     --source-kube-clustername api-test-example-com:6443
I0302 09:45:19.026520 4181483 options.go:305] ReplicationSource source-scribe-source created in namespace source
```

### Sync an SSH secret from the destination namespace to the source namespace

This assumes the default secret name that is created by the scribe controller. You can also pass `--ssh-keys-secret`
that is a valid ssh-key-secret in the DestinationReplication namespace and cluster.

_You may omit the clustername, context flags for whichever context is the current context_

```bash
scribe sync-ssh-secret \
     --dest-namespace dest \
     --dest-kube-clustername test0123 --dest-kube-context testuser \
     --source-namespace source \
     --source-kube-clustername test034 --source-kube-context admin
```

TODO: add this to scribe CLI
### Finally, create a database to sync in the destination ns

Find the latest image from the ReplicationDestination, then
use this image to create the PVC

```bash
$ kubectl --context testuser get replicationdestination dest-scribe-destination -n dest --template={{.status.latestImage.name}}
$ sed -i 's/snapshotToReplace/scribe-dest-database-destination-20201203174504/g' ../scribe/examples/destination-database/mysql-pvc.yaml
$ kubectl --context testuser apply -n dest -f ../scribe/examples/destination-database/
```
