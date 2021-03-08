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
	scribeNewDestinationLong = templates.LongDesc(`
Scribe is a command line tool for a scribe operator running in a Kubernetes cluster. Scribe asynchronously replicates Kubernetes persistent volumes between clusters or namespaces using rsync, rclone, or restic. Scribe uses a ReplicationDestination and a ReplicationSource to replicate a volume. Data will be synced according to the configured sync schedule.
`)
	scribeNewDestinationExample = templates.Examples(`
        # Create a ReplicationDestination in the namespace 'dest'.
        scribe new-destination --dest-namespace dest --dest-copy-method Snapshot --dest-access-mode ReadWriteOnce

        # Create a ReplicationDestination in the current namespace that will use Snapshot copy method and existing pvc mysql-claim
        scribe new-destination  --dest-copy-method Snapshot --dest-pvc mysql-claim

		# Create a ReplicationDestination in the namespace 'dest' in cluster 'api-test-test-com:6443' with context 'scribe-user'.
        scribe new-destination --dest-namespace dest \
		    --dest-copy-method Snapshot --dest-access-mode ReadWriteOnce \
			--dest-kube-context scribe-user --dest-kube-clustername api-test-test-com:6443
    `)
)

func NewCmdScribeNewDestination(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewReplicationOptions(streams)
	cmd := &cobra.Command{
		Use:     "new-destination [OPTIONS]",
		Short:   i18n.T("Create a ReplicationDestination for replicating a persistent volume."),
		Long:    fmt.Sprintf(scribeNewDestinationLong),
		Example: fmt.Sprintf(scribeNewDestinationExample),
		Run: func(cmd *cobra.Command, args []string) {
			kcmdutil.CheckErr(o.Complete(cmd))
			kcmdutil.CheckErr(o.Validate())
			kcmdutil.CheckErr(o.CreateReplicationDestination())
		},
	}
	flags := cmd.Flags()
	o.scribeOptions.Bind(flags)
	o.sshKeysSecretOptions.Bind(flags)
	flags.StringVar(&o.Mode, "mode", "destination", "to distinguish destination options from source options")
	flags.StringVar(&o.CopyMethod, "dest-copy-method", o.CopyMethod, "the method of creating a point-in-time image of the destination volume; one of 'None|Clone|Snapshot'")
	flags.StringVar(&o.Address, "address", o.Address, "the remote address to connect to for replication.")
	// TODO: Defaulted with CLI, should it be??
	flags.StringVar(&o.Capacity, "dest-capacity", "2Gi", "Size of the destination volume to create. Must be provided if --dest-pvc is not provided.")
	flags.StringVar(&o.StorageClassName, "dest-storage-class-name", o.StorageClassName, "name of the StorageClass of the destination volume. If not set, the default StorageClass will be used.")
	flags.StringVar(&o.AccessMode, "dest-access-mode", o.AccessMode, "the access modes for the destination volume. Must be provided if --dest-pvc is not provided; One of 'ReadWriteOnce|ReadOnlyMany|ReadWriteMany")
	flags.StringVar(&o.VolumeSnapshotClassName, "dest-volume-snapshot-class", o.VolumeSnapshotClassName, "name of the VolumeSnapshotClass to be used for the destination volume, only if the copyMethod is 'Snapshot'. If not set, the default VSC will be used.")
	flags.StringVar(&o.PVC, "dest-pvc", o.PVC, "name of an existing PVC to use as the transfer destination volume instead of automatically provisioning one.")
	flags.StringVar(&o.Schedule, "dest-cron-spec", o.Schedule, "cronspec to be used to schedule replication to occur at regular, time-based intervals. If not set replication will be continuous.")
	// Defaults to "root" after creation
	flags.StringVar(&o.SSHUser, "dest-ssh-user", o.SSHUser, "username for outgoing SSH connections (default 'root')")
	// Defaults to ClusterIP after creation
	flags.StringVar(&o.ServiceType, "dest-service-type", o.ServiceType, "one of ClusterIP|LoadBalancer. Service type to be created for incoming SSH connections. (default 'ClusterIP')")
	// TODO: Defaulted in CLI, should it be??
	flags.StringVar(&o.Name, "dest-name", o.Name, "name of the ReplicationDestination resource. (default '<current-namespace>-scribe-destination')")
	// defaults to 22 after creation
	flags.Int32Var(&o.Port, "port", o.Port, "SSH port to connect to for replication (default 22)")
	flags.StringVar(&o.Provider, "provider", o.Provider, "name of an external replication provider, if applicable; pass as 'domain.com/provider'")
	// TODO: I don't know how many params providers have? If a lot, can pass a file instead
	flags.StringVar(&o.ProviderParameters, "dest-provider-parameters", o.ProviderParameters, "provider-specific key/value configuration parameters, if using an external provider; pass as 'key/value,key1/value1,key2/value2'")
	// defaults to "/" after creation
	flags.StringVar(&o.Path, "path", o.Path, "the remote path to rsync from. (default '/')")
	flags.MarkHidden("mode")
	cmd.MarkFlagRequired("dest-copy-method")

	return cmd
}
