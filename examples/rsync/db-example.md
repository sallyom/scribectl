# Rsync Database Example

This example will sync data from mysql database persistent volumes
For this example, sync will happen within a single cluster and 2 namespaces.

***https://github.com/backube/scribe checked out at ../scribe***

### First, create a replication destination:

```console
$ kubectl create ns dest
$ ./scribe new-destination --dest-namespace dest --dest-access-mode ReadWriteOnce --dest-copy-method Snapshot 
I0302 09:28:35.028745 4174293 options.go:248] ReplicationDestination dest-scribe-destination created in namespace dest
$ address=$(kubectl get replicationdestination/dest-scribe-destination  -n dest --template={{.status.rsync.address}})
```
### Now, create source application with:

```console
$ kubectl create ns src
$ kubectl -n src apply -f ../scribe/examples/source-database/
```

### Create a database or other changes in the mysql database:

```console
$ kubectl exec --stdin --tty -n source `kubectl get pods -n src | grep mysql | awk '{print $1}'` -- /bin/bash
$ mysql -u root -p$MYSQL_ROOT_PASSWORD
> create database my_new_database;
> exit
$ exit
```

### Now, create replication source:

```console
$ ./scribe new-source --address ${address} --source-namespace src --source-copy-method Snapshot --source-pvc mysql-pv-claim
I0302 09:45:19.026520 4181483 options.go:305] ReplicationSource src-scribe-source created in namespace src
```

TODO: add this to scribe CLI
### Obtain an SSH secret from the destination namespace 

```console
$ kubectl get secret -n dest scribe-rsync-dest-src-dest-scribe-destination -o yaml > /tmp/secret.yaml
$ vi /tmp/secret.yaml
# ^^^ change the namespace to "src"
# ^^^ remove the owner reference (.metadata.ownerReferences)
$ kubectl apply -f /tmp/secret.yaml
```

TODO: add this to scribe CLI
### Finally, create a database to sync in the destination ns

Find the latest image from the ReplicationDestination, then
use this image to create the PVC

```console
$ kubectl get replicationdestination dest-scribe-destination -n dest --template={{.status.latestImage.name}}
$ sed -i 's/snapshotToReplace/scribe-dest-database-destination-20201203174504/g' ../scribe/examples/destination-database/mysql-pvc.yaml
$ kubectl apply -n dest -f ../scribe/examples/destination-database/
```
