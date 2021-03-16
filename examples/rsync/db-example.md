# Rsync Database Example

This example will sync data from mysql database persistent volumes
For this example, sync will happen within a single cluster and 2 namespaces.

*  **The cluster must have the [scribe operator installed](https://scribe-replication.readthedocs.io/en/latest/installation/index.html)**
*  ***https://github.com/backube/scribe checked out at ../scribe***

### Build Scribe CLI

```bash
$ make scribe
$ mv scribe /usr/local/bin (or add to $PATH)
```

### Create a scribe-config with necessary flags:

Create a config file to designate your source and destination options. You can also pass these individually to each command, but they add up so the
config file is usually a good option. You can add any, some, or all flags from `scribe <command> --help` to the config file.

Create the config file at `./scribe-config`, as scribe will look for that file in the current directory.
These are the flags that can always be filled in before creating either destination or source. You can change the values to suit your needs.

```bash
$ cat scribe-config
dest-access-mode: ReadWriteOnce
dest-copy-method: Snapshot
dest-namespace: dest
source-namespace: source
source-pvc: mysql-pv-claim
source-copy-method: Snapshot
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

Necessary flags are configured in `./scribe-config` shown above.
```bash
$ kubectl create ns dest
$ scribe new-destination
I0302 09:28:35.028745 4174293 options.go:248] ReplicationDestination dest-destination created in namespace dest
$ address=$(kubectl get replicationdestination/dest-destination  -n dest --template={{.status.rsync.address}})
$ echo ${address} //to be sure it's not empty, may take a minute to populate
```

### Sync an SSH secret from the destination namespace to the source namespace

This assumes the default secret name that is created by the scribe controller. You can also pass `--ssh-keys-secret`
that is a valid ssh-key-secret in the DestinationReplication namespace and cluster.

Necessary flags are configured in `./scribe-config` shown above.  Save the output from the command below,
as you will need the name of the ssh-keys-secret to pass to `scribe new-source`.
```bash
scribe sync-ssh-secret 
```

### Create a replication source:

Necessary flags are configured in `./scribe-config` shown above.
```bash
$ scribe new-source --address ${address} --ssh-keys-secret <name-of-ssh-secret-from-output-of-sync>
I0302 09:45:19.026520 4181483 options.go:305] ReplicationSource source-scribe-source created in namespace source
```
TODO: add this to scribe CLI
### Finally, create a database to sync in the destination namespace

First, create the destination application from the scribe example:
```bash
$ kubectl apply -n dest -f ../scribe/examples/destination-database/
$ kubectl get pvc/mysql-pv-claim -n dest -o yaml > /tmp/pvc.yaml
```

To sync the data, you have to replace the PVC (and PV). This is because PersistenVolumeClaims are immutable.
That is the reason for creating the PVC, extracting the yaml to a local file, then updating the snapshot image.
For each sync, find the latest image from the ReplicationDestination, then use this image to create the PVC

The following steps can be repeated to sync the data from source to destination:
```bash
$ kubectl delete pvc/mysql-pv-claim -n dest --force --grace-period=0
$ SNAPSHOT=$(kubectl get replicationdestination dest-destination -n dest --template={{.status.latestImage.name}})
$ echo ${SNAPSHOT} // make sure this is not empty
$ sed -i "s/snapshotToReplace/${SNAPSHOT}/g" /tmp/pvc.yaml
$ kubectl apply -f /tmp/pvc.yaml

For the next sync, reset the pvc.yaml
$ sed -i "s/${SNAPSHOT}/snapshotToReplace/g" /tmp/pvc.yaml
```

Verify the synced database:
```bash
$ kubectl exec --stdin --tty -n dest `kubectl get pods -n dest | grep mysql | awk '{print $1}'` -- /bin/bash
# mysql -u root -p$MYSQL_ROOT_PASSWORD
> show databases;
> exit
$ exit
```
