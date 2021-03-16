# Multiple Cluster/Multiple Kubeconfig Contexts Rsync Database Example

This example will sync data from mysql database persistent volumes
For this example, sync will happen between 2 clusters. Data will be synced
from cluster-name `api-source-com:6443` to cluster-name `destination123`

*  **Both clusters must have the [scribe operator installed](https://scribe-replication.readthedocs.io/en/latest/installation/index.html)**
*  ***https://github.com/backube/scribe checked out at ../scribe***

### Build Scribe

```bash
$ make scribe
$ mv scribe /usr/local/bin (or add to $PATH)
```

### Merge Kubeconfigs (if clusters not already in a single kubeconfig)

~/kubeconfig1 with context `destuser` and cluster-name `destination123`

~/kubeconfig2 with context `sourceuser` and cluster-name `api-source-com:6443`
```bash
$ export KUBECONFIG=~/kubeconfig1:~/kubeconfig2

You can view config with the following commands:
$ kubectl config view
$ kubectl config get-clusters
$ kubectl config get-contexts

You can rename contexts with the following:
$ kubectl config rename-context <oldname> <newname>
```

### Create source application:

```bash
$ kubectl --context sourceuser create ns source
$ kubectl --context sourceuser -n source apply -f ../scribe/examples/source-database/
```

### Create a database or other changes in the mysql database:

```bash
$ kubectl --context sourceuser exec --stdin --tty -n source `kubectl --context sourceuser get pods -n source | grep mysql | awk '{print $1}'` -- /bin/bash
$ mysql -u root -p$MYSQL_ROOT_PASSWORD
> create database my_new_database;
> exit
$ exit
```

### Create a scribe-config with necessary flags:

Create a config file to designate your source and destination options. You can also pass these individually to each command, but they add up so the
config file is usually a good option. You can add any, some, or all flags from `scribe <command> --help` to the config file.

Create the config file at `./scribe-config`, as scribe will look for that file in the current directory.
These are the flags that can always be filled in before creating either destination or source. You can change the values to suit your needs.

```bash
$ cat scribe-config
dest-kube-context: destuser
dest-kube-clustername: destination123
dest-service-type: LoadBalancer
dest-access-mode: ReadWriteOnce
dest-copy-method: Snapshot
dest-namespace: dest
source-kube-context: sourceuser
source-kube-clustername: api-source-com:6443
source-namespace: source
source-service-type: LoadBalancer
source-copy-method: Snapshot
```

### Create a replication destination:

Necessary flags are configured in `./scribe-config` shown above.
```bash
$ kubectl --context destuser create ns dest
$ scribe new-destination
I0302 09:28:35.028745 4174293 options.go:248] ReplicationDestination dest-destination created in namespace dest
```
Save the rsync address from the destination to pass to the new-source:
```bash
$ address=$(kubectl --context destuser get replicationdestination/dest-destination  -n dest --template={{.status.rsync.address}})
$ echo ${address} //to be sure it's not empty, may take a minute to populate
```

### Sync an SSH secret from the destination namespace to the source namespace

This assumes the default secret name that is created by the scribe controller. You can also pass `--ssh-keys-secret`
that is a valid ssh-key-secret in the DestinationReplication namespace and cluster.

Necessary flags are configured in `./scribe-config` shown above.
Save the output from the command below, as you will need the name of the ssh-keys-secret to pass to `scribe new-source`

```bash
scribe sync-ssh-secret
```

### Create replication source:

Necessary flags are configured in `./scribe-config` shown above.
The ssh-keys-secret name listed below is the default secret name that is created from `scribe sync-ssh-secret`.

```bash
$ scribe new-source --address ${address} --ssh-keys-secret scribe-rsync-dest-src-<name-of-replicationdestination> 
I0302 09:45:19.026520 4181483 options.go:305] ReplicationSource source-scribe-source created in namespace source
```

For the rest of the example, you'll be working from the `destuser context`. So we don't have to pass that to every
kubectl command, run this:
```bash
$ kubectl config use-context destuser
```

TODO: add this to scribe CLI
### Finally, create a database to sync in the destination namespace

First, create the destination application from the scribe example:
```bash
$ cp ../scribe/examples/destination-database/mysql-pvc.yaml /tmp/pvc.yaml // will use that later
# - edit the /tmp/pvc.yaml with metadata.namespace `dest` directly under the `name:`, otherwise you 
#   may forget to add the `-n dest` when you apply the yaml (like I did).
$ kubectl apply -n dest -f ../scribe/examples/destination-database/mysql-deployment.yaml
$ kubectl apply -n dest -f ../scribe/examples/destination-database/mysql-service.yaml
$ kubectl apply -n dest -f ../scribe/examples/destination-database/mysql-secret.yaml
```

To sync the data, you have to replace the PVC each time. This is because PersistenVolumeClaims are immutable.
That is the reason for creating the PVC, extracting the yaml to a local file, then updating the snapshot image.
For each sync, find the latest image from the ReplicationDestination, then use this image to create the PVC

For first data sync run these commands:
```bash
$ SNAPSHOT=$(kubectl get replicationdestination dest-destination -n dest --template={{.status.latestImage.name}})
$ echo ${SNAPSHOT} // make sure this is not empty
$ sed -i "s/snapshotToReplace/${SNAPSHOT}/g" /tmp/pvc.yaml
$ kubectl apply -f /tmp/pvc.yaml
```

Verify the synced database:
```bash
$ kubectl exec --stdin --tty -n dest `kubectl get pods -n dest | grep mysql | awk '{print $1}'` -- /bin/bash
# mysql -u root -p$MYSQL_ROOT_PASSWORD
> show databases;
> exit
$ exit
```

### Pattern to follow for all future syncs - It's not pretty but it works:

To sync the data, you have to replace the PVC with every sync. This is because PersistenVolumeClaims are immutable.

You will also need to update the PV for the PVC with each sync. To do that,
find the PV that is bound to the mysql-pv-claim PVC, then remove its claimRef:
```bash
$ PVNAME=$(kubectl get pv | grep mysql-pv-claim | awk '{print $1}')
$ kubectl delete pvc/mysql-pv-claim -n dest // This is tricky, this will hang, when it does, 
  ctrl-c to escape the cmd, then:
$ kubectl edit pvc/mysql-pv-claim -n dest // Remove the finalizer section. Once removed save and exit,
  since the pvc has a deletionTimestamp it will disappear when you remove the finalizer.
$ kubectl patch pv "${PVNAME}" -p '{"spec":{"claimRef": null}}'

Now the PV is ready to bind to a new pvc/mysql-pv-claim with a new snapshot.
```

Reset the /tmp/pvc.yaml:
```bash
$ sed -i "s/${SNAPSHOT}/snapshotToReplace/g" /tmp/pvc.yaml
```

(One-time-only step): Edit the /tmp/pvc.yaml to add the PV Name. Add the spec.volumeName like so:
```
spec:
  volumeName: ${PVNAME}
  accessModes:
    - ReadWriteOnce
----
```

Now, you can follow the same steps as above to get the new snapshot, update the pvc.yaml with it, and apply the new destination pvc:

```bash
$ SNAPSHOT=$(kubectl get replicationdestination dest-destination -n dest --template={{.status.latestImage.name}})
$ echo ${SNAPSHOT} // make sure this is not empty
$ sed -i "s/snapshotToReplace/${SNAPSHOT}/g" /tmp/pvc.yaml
$ kubectl apply -f /tmp/pvc.yaml
```

Verify the synced database.

