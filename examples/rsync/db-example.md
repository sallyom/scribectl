# Rsync Database Example

This example will sync data from mysql database persistent volumes
For this example, sync will happen within a single cluster and 2 namespaces.
**The cluster must have the [scribe operator installed](https://scribe-replication.readthedocs.io/en/latest/installation/index.html)**

***https://github.com/backube/scribe checked out at ../scribe***

### Build Scribe CLI

```bash
$ make scribe
$ mv scribe /usr/local/bin (or add to $PATH)
```

### Create source application with:

```bash
$ kubectl create ns source
$ kubectl -n source apply -f ../scribe/examples/source-database/
```

### Create a database or other changes in the mysql database:

```bash
$ kubectl exec --stdin --tty -n source `kubectl get pods -n source | grep mysql | awk '{print $1}'` -- /bin/bash
$ mysql -u root -p$MYSQL_ROOT_PASSWORD
> create database my_new_database;
> exit
$ exit
```

### Create a replication destination:

```bash
$ kubectl create ns dest
$ ./scribe new-destination --dest-namespace dest --dest-access-mode ReadWriteOnce --dest-copy-method Snapshot 
I0302 09:28:35.028745 4174293 options.go:248] ReplicationDestination dest-scribe-destination created in namespace dest
$ address=$(kubectl get replicationdestination/dest-scribe-destination  -n dest --template={{.status.rsync.address}})
```

### Sync an SSH secret from the destination namespace to the source namespace

This assumes the default secret name that is created by the scribe controller. You can also pass `--ssh-keys-secret`
that is a valid ssh-key-secret in the DestinationReplication namespace and cluster.

```bash
scribe sync-ssh-secret --dest-namespace dest --source-namespace source 
```

### Create a replication source:

```bash
$ ./scribe new-source --address ${address} --source-namespace source \
    --source-copy-method Snapshot --source-pvc mysql-pv-claim \
    --ssh-keys-secret scribe-rsync-dest-src-<name-of-replicationdestination>
I0302 09:45:19.026520 4181483 options.go:305] ReplicationSource source-scribe-source created in namespace source
```
TODO: add this to scribe CLI
### Finally, create a database to sync in the destination ns

Find the latest image from the ReplicationDestination, then
use this image to create the PVC

```bash
$ kubectl get replicationdestination dest-scribe-destination -n dest --template={{.status.latestImage.name}}
$ sed -i 's/snapshotToReplace/scribe-dest-database-destination-20201203174504/g' ../scribe/examples/destination-database/mysql-pvc.yaml
$ kubectl apply -n dest -f ../scribe/examples/destination-database/
```

Verify the synced database:
```bash
$ kubectl exec --stdin --tty -n dest `kubectl get pods -n dest | grep mysql | awk '{print $1}'` -- /bin/bash
# mysql -u root -p$MYSQL_ROOT_PASSWORD
> show databases;
> exit
$ exit
```
