package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	kcmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/i18n"
	"k8s.io/kubectl/pkg/util/templates"
)

var (
	scribeNewSourceLong = templates.LongDesc(`
Scribe is a command line tool for a scribe operator running in a Kubernetes cluster. Scribe asynchronously replicates Kubernetes persistent volumes between clusters or namespaces using rsync, rclone, or restic. Scribe uses a ReplicationDestination and a ReplicationSource to replicate a volume. Data will be synced according to the configured sync schedule.
`)
	scribeNewSourceExample = templates.Examples(`
        # Create a ReplicationSource for mysql-pvc using Snapshot copy method in the namespace 'source'.
        $ scribe new-source --source-namespace source --source-copy-method Snapshot --source-pvc mysql-pvc

        # Create a ReplicationSource for mysql-pvc using Snapshot copy method in the namespace 'source'
		# in clustername 'api-source-test-com:6443' with context 'user-scribe'.
        $ scribe new-source --source-namespace source \
		    --source-copy-method Snapshot --source-pvc mysql-pvc \
			--source-kube-context user-scribe --source-clustername api-source-test-com:6443

        # Create a ReplicationSource for mysql-pvc using Clone copy method in the current namespace.
        $ scribe new-source --source-copy-method Clone --source-pvc mysql-pvc
    `)
)

func NewCmdScribeNewSource(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewReplicationOptions(streams)
	cmd := &cobra.Command{
		Use:     "new-source [OPTIONS]",
		Short:   i18n.T("Create a ReplicationSource for replicating a persistent volume."),
		Long:    fmt.Sprintf(scribeNewSourceLong),
		Example: fmt.Sprintf(scribeNewSourceExample),
		Run: func(cmd *cobra.Command, args []string) {
			kcmdutil.CheckErr(o.Complete(cmd))
			kcmdutil.CheckErr(o.Validate())
			kcmdutil.CheckErr(o.CreateReplicationSource())
		},
	}
	flags := cmd.Flags()
	o.scribeOptions.Bind(flags)
	flags.StringVar(&o.Mode, "mode", "source", "to distinguish destination options from source options")
	flags.StringVar(&o.CopyMethod, "source-copy-method", o.CopyMethod, "the method of creating a point-in-time image of the source volume; one of 'None|Clone|Snapshot'")
	flags.StringVar(&o.Address, "address", o.Address, "the remote address to connect to for replication.")
	flags.StringVar(&o.Capacity, "source-capacity", o.Capacity, "provided to override the capacity of the point-in-Time image.")
	flags.StringVar(&o.StorageClassName, "source-storage-class-name", o.StorageClassName, "provided to override the StorageClass of the point-in-Time image.")
	flags.StringVar(&o.AccessMode, "source-access-mode", o.AccessMode, "provided to override the accessModes of the point-in-Time image. One of 'ReadWriteOnce|ReadOnlyMany|ReadWriteMany")
	flags.StringVar(&o.VolumeSnapshotClassName, "source-volume-snapshot-class", o.VolumeSnapshotClassName, "name of the VolumeSnapshotClass to be used for the source volume, only if the copyMethod is 'Snapshot'. If not set, the default VSC will be used.")
	flags.StringVar(&o.PVC, "source-pvc", o.PVC, "name of an existing PersistentVolumeClaim (PVC) to replicate.")
	flags.StringVar(&o.Namespace, "source-namespace", o.Namespace, "the transfer source namespace. This namespace must exist.")
	// TODO: Default to every 3min for source?
	flags.StringVar(&o.Schedule, "source-cron-spec", "*/3 * * * *", "cronspec to be used to schedule capturing the state of the source volume. If not set the source volume will be captured every 3 minutes.")
	// TODO: should this be exposed?
	flags.StringVar(&o.SSHKeys, "source-ssh-keys", o.SSHKeys, "name of a secret in the source namespace to be used for authentication. If not set, SSH keys will be generated and a secret will be created with the appropriate keys.")
	// Defaults to "root" after creation
	flags.StringVar(&o.SSHUser, "source-ssh-user", o.SSHUser, "username for outgoing SSH connections (default 'root')")
	// Defaults to ClusterIP after creation
	flags.StringVar(&o.ServiceType, "source-service-type", o.ServiceType, "the Service type that will be created for incoming SSH connections. (default 'ClusterIP')")
	// TODO: Defaulted in CLI, should it be??
	flags.StringVar(&o.Name, "source-name", o.Name, "name of the ReplicationSource resource (default '<source-ns>-scribe-source')")
	// defaults to 22 after creation
	flags.Int32Var(&o.Port, "port", o.Port, "SSH port to connect to for replication. (default 22)")
	flags.StringVar(&o.Provider, "provider", o.Provider, "name of an external replication provider, if applicable; pass as 'domain.com/provider'")
	// TODO: I don't know how many params providers have? If a lot, can pass a file instead
	flags.StringVar(&o.ProviderParameters, "provider-parameters", o.ProviderParameters, "provider-specific key/value configuration parameters, if using an external provider; pass as 'key/value,key1/value1,key2/value2'")
	// defaults to "/" after creation
	flags.StringVar(&o.Path, "path", o.Path, "the remote path to rsync to (default '/')")
	flags.MarkHidden("mode")
	cmd.MarkFlagRequired("source-copy-method")
	cmd.MarkFlagRequired("source-pvc")

	return cmd
}
