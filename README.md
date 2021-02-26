
# Coming Soon: scribectl
command line interface for backube/scribe

***This tool is currently being developed, only `new-destination` & `new-source` exists now, check back in coming weeks***

# Scribe

Scribe asynchronously replicates Kubernetes persistent volumes between clusters
using either rsync or rclone depending on the number of destinations.

## Try Scribe in Kind

To try out Scribe in a `kind cluster`, follow the steps in [scribe/hack/run-in-kind.sh](https://github.com/backube/scribe/blob/master/hack/run-in-kind.sh).


## To install Scribe in a Kubernetes or OpenShift cluster:

To try out Scribe,  follow the steps in the [installation
instructions](https://scribe-replication.readthedocs.io/en/latest/installation/index.html).

## Tips for setting up storage in a cluster (already included in the run-in-kind script):

### AWS tips to set up your storage to use Snapshot CopyMethod:

```console
# Switch default StorageClass to be the EBS CSI driver

$ kubectl annotate sc/gp2 storageclass.kubernetes.io/is-default-class="false" --overwrite
$ kubectl annotate sc/gp2-csi storageclass.kubernetes.io/is-default-class="true" --overwrite

# Install a VolumeSnapshotClass
$ kubectl create -f - << SNAPCLASS
---
apiVersion: snapshot.storage.k8s.io/v1beta1
kind: VolumeSnapshotClass
metadata:
  name: gp2-csi
driver: ebs.csi.aws.com
deletionPolicy: Delete
SNAPCLASS

# Set gp2-csi as default VolumeSnapshotClass
$ kubectl annotate volumesnapshotclass/gp2-csi snapshot.storage.kubernetes.io/is-default-class="true"
```

### Before using scribectl, run through this example to become familar:
Try out Scribe with [this rsync example](https://github.com/backube/scribe/blob/master/docs/usage/rsync/database_example.rst)!

TODO: Replace example above with CLI commands

```
Now you're ready to try scribectl!
```
