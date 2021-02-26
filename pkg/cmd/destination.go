package cmd

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	kcmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/i18n"
	"k8s.io/kubectl/pkg/util/templates"
)

var (
	scribeNewDestinationLong = templates.LongDesc(`
        Scribe is a command line tool for a scribe operator running in a Kubernetes cluster. Scribe asynchronously replicates Kubernetes persistent volumes between clusters or namespaces
		using rsync, rclone, or restic. Scribe uses a ReplicationDestination and a ReplicationSource to replicate a volume. Data will be synced according to the configured sync schedule.
	`)
	scribeNewDestinationExample = templates.Examples(`
        # Create a ReplicationDestination in the namespace 'dest'.
        scribe new-destination --dest-namespace dest --dest-copy-method Snapshot --dest-access-mode ReadWriteOnce

        # Create a ReplicationDestination in the namespace 'dest' that will use Snapshot copy method with ReadWriteOnce access mode.
        scribe new-destination --dest-namespace dest --dest-access-mode ReadWriteOnce --dest-copy-method Snapshot

        # Create a ReplicationDestination in the current namespace that will use Snapshot copy method and existing pvc mysql-claim
        scribe new-destination  --dest-copy-method Snapshot --dest-pvc mysql-claim
    `)
)

func NewCmdScribeNewDestination(f kcmdutil.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	o := NewReplicationOptions(streams)
	cmd := &cobra.Command{
		Use:     "new-destination [OPTIONS]",
		Short:   i18n.T("Create a ReplicationDestination for replicating a persistent volume."),
		Long:    scribeNewDestinationLong,
		Example: scribeNewDestinationExample,
		Run: func(cmd *cobra.Command, args []string) {
			kcmdutil.CheckErr(o.Complete(f, cmd))
			kcmdutil.CheckErr(o.Validate())
			kcmdutil.CheckErr(o.CreateReplicationDestination())
		},
	}
	cmd.Flags().StringVar(&o.Mode, "mode", "destination", "to distinguish destination options from source options")
	cmd.Flags().StringVar(&o.CopyMethod, "dest-copy-method", o.CopyMethod, "the method of creating a point-in-time image of the destination volume; one of 'None|Clone|Snapshot'")
	cmd.Flags().StringVar(&o.Address, "address", o.Address, "the remote address to connect to for replication.")
	// TODO: Defaulted with CLI, should it be??
	cmd.Flags().StringVar(&o.Capacity, "dest-capacity", "2Gi", "(2Gi) Size of the destination volume to create. Must be provided if --dest-pvc is not provided.")
	cmd.Flags().StringVar(&o.StorageClassName, "dest-storage-class-name", o.StorageClassName, "the name of the StorageClass of the destination volume. If not set, the default StorageClass will be used.")
	cmd.Flags().StringVar(&o.AccessMode, "dest-access-mode", o.AccessMode, "the access modes for the destination volume. Must be provided if --dest-pvc is not provided; One of 'ReadWriteOnce|ReadOnlyMany|ReadWriteMany")
	cmd.Flags().StringVar(&o.VolumeSnapshotClassName, "dest-volume-snapshot-class", o.VolumeSnapshotClassName, "the name of the VolumeSnapshotClass to be used for the destination volume, only if the copyMethod is 'Snapshot'. If not set, the default VSC will be used.")
	cmd.Flags().StringVar(&o.PVC, "dest-pvc", o.PVC, "the name of an existing PVC to use as the transfer destination volume instead of automatically provisioning one.")
	cmd.Flags().StringVar(&o.Namespace, "dest-namespace", o.Namespace, "the transfer destination namespace. This namespace must exist. If not set, the ReplicationDestination will be created in the current namespace.")
	// TODO: Default?
	cmd.Flags().StringVar(&o.Schedule, "dest-cron-spec", o.Schedule, "cronspec to be used to schedule replication to occur at regular, time-based intervals. If not set replication will be continuous.")
	// TODO: should this be exposed?
	cmd.Flags().StringVar(&o.SSHKeys, "dest-ssh-keys", o.SSHKeys, "name of a secret in the destination namespace to be used for authentication. If not set, SSH keys will be generated and a secret will be created with the appropriate keys.")
	// Defaults to "root" after creation
	cmd.Flags().StringVar(&o.SSHUser, "dest-ssh-user", o.SSHUser, "(root) username for outgoing SSH connections")
	// Defaults to ClusterIP after creation
	cmd.Flags().StringVar(&o.ServiceType, "dest-service-type", o.ServiceType, "(ClusterIP) Service type to be created for incoming SSH connections.")
	// TODO: Defaulted in CLI, should it be??
	cmd.Flags().StringVar(&o.Name, "dest-name", o.Name, "(<namespace>-scribe-destination) name of the ReplicationDestination resource")
	// defaults to 22 after creation
	cmd.Flags().Int32Var(&o.Port, "port", o.Port, "(22) SSH port to connect to for replication")
	cmd.Flags().StringVar(&o.Provider, "provider", o.Provider, "name of an external replication provider, if applicable; pass as 'domain.com/provider'")
	// TODO: I don't know how many params providers have? If a lot, can pass a file instead
	cmd.Flags().StringVar(&o.ProviderParameters, "dest-provider-parameters", o.ProviderParameters, "provider-specific key/value configuration parameters, if using an external provider; pass as 'key/value,key1/value1,key2/value2'")
	// defaults to "/" after creation
	cmd.Flags().StringVar(&o.Path, "path", o.Path, "(/) the remote path to rsync from")
	cmd.Flags().MarkHidden("mode")
	cmd.MarkFlagRequired("dest-copy-method")

	return cmd
}
