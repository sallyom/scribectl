package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	kcmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/i18n"
	"k8s.io/kubectl/pkg/util/templates"
	"k8s.io/klog/v2"

	scribev1alpha1 "github.com/backube/scribe/api/v1alpha1"
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

type destinationOptions struct {
	scribeOptions               scribeOptions
	sshKeysSecretOptions        sshKeysSecretOptions
	DestCopyMethod              string //v1alpha1.CopyMethodType
	DestCapacity                string //*resource.Quantity
	DestStorageClassName        string
	DestAccessMode              string //[]corev1.PersistentVolumeAccessMode
	Address                     string
	DestVolumeSnapshotClassName string
	DestPVC                     string
	DestSchedule                string
	SSHUser                     string
	DestName                    string
	DestNamespace               string
	DestServiceType             string //*corev1.ServiceType
	Port                        int32  //int32
	Path                        string
	RcloneConfig                string
	Provider                    string
	ProviderParameters          string //map[string]string
	genericclioptions.IOStreams
}

func NewDestinationOptions(streams genericclioptions.IOStreams) *destinationOptions {
	return &destinationOptions{
		IOStreams: streams,
	}
}

func NewCmdScribeNewDestination(streams genericclioptions.IOStreams) *cobra.Command {
	v := viper.New()
	o := NewDestinationOptions(streams)
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
	kcmdutil.CheckErr(o.scribeOptions.Bind(cmd, v))
	kcmdutil.CheckErr(o.sshKeysSecretOptions.Bind(cmd, v))
	kcmdutil.CheckErr(o.Bind(cmd, v))

	return cmd
}

func (o *destinationOptions) bindFlags(cmd *cobra.Command, v *viper.Viper) error {
	flags := cmd.Flags()
	flags.StringVar(&o.DestCopyMethod, "dest-copy-method", o.DestCopyMethod, "the method of creating a point-in-time image of the destination volume; one of 'None|Clone|Snapshot'")
	flags.StringVar(&o.Address, "address", o.Address, "the remote address to connect to for replication.")
	// TODO: Defaulted with CLI, should it be??
	flags.StringVar(&o.DestCapacity, "dest-capacity", "2Gi", "Size of the destination volume to create. Must be provided if --dest-pvc is not provided.")
	flags.StringVar(&o.DestStorageClassName, "dest-storage-class-name", o.DestStorageClassName, "name of the StorageClass of the destination volume. If not set, the default StorageClass will be used.")
	flags.StringVar(&o.DestAccessMode, "dest-access-mode", o.DestAccessMode, "the access modes for the destination volume. Must be provided if --dest-pvc is not provided; One of 'ReadWriteOnce|ReadOnlyMany|ReadWriteMany")
	flags.StringVar(&o.DestVolumeSnapshotClassName, "dest-volume-snapshot-class", o.DestVolumeSnapshotClassName, "name of the VolumeSnapshotClass to be used for the destination volume, only if the copyMethod is 'Snapshot'. If not set, the default VSC will be used.")
	flags.StringVar(&o.DestPVC, "dest-pvc", o.DestPVC, "name of an existing PVC to use as the transfer destination volume instead of automatically provisioning one.")
	flags.StringVar(&o.DestSchedule, "dest-cron-spec", o.DestSchedule, "cronspec to be used to schedule replication to occur at regular, time-based intervals. If not set replication will be continuous.")
	// Defaults to "root" after creation
	flags.StringVar(&o.SSHUser, "dest-ssh-user", o.SSHUser, "username for outgoing SSH connections (default 'root')")
	// Defaults to ClusterIP after creation
	flags.StringVar(&o.DestServiceType, "dest-service-type", o.DestServiceType, "one of ClusterIP|LoadBalancer. Service type to be created for incoming SSH connections. (default 'ClusterIP')")
	// TODO: Defaulted in CLI, should it be??
	flags.StringVar(&o.DestName, "dest-name", o.DestName, "name of the ReplicationDestination resource. (default '<current-namespace>-scribe-destination')")
	flags.Int32Var(&o.Port, "port", o.Port, "SSH port to connect to for replication. (default 22)")
	flags.StringVar(&o.Provider, "provider", o.Provider, "name of an external replication provider, if applicable; pass as 'domain.com/provider'")
	// TODO: I don't know how many params providers have? If a lot, can pass a file instead
	flags.StringVar(&o.ProviderParameters, "provider-parameters", o.ProviderParameters, "provider-specific key/value configuration parameters, if using an external provider; pass as 'key/value,key1/value1,key2/value2'")
	// defaults to "/" after creation
	flags.StringVar(&o.Path, "path", o.Path, "the remote path to rsync to (default '/')")
	cmd.MarkFlagRequired("dest-copy-method")
	flags.VisitAll(func(f *pflag.Flag) {
		if !f.Changed && v.IsSet(f.Name) {
			val := v.Get(f.Name)
			flags.Set(f.Name, fmt.Sprintf("%v", val))
		}
	})
	return nil
}

func (o *destinationOptions) Bind(cmd *cobra.Command, v *viper.Viper) error {
	// config file in current directory
	// TODO: where to look for config file
	v.SetConfigName(scribeConfig)
	v.AddConfigPath(".")
	v.SetConfigType("yaml")
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}
	o.bindFlags(cmd, v)
	return nil
}

func (o *destinationOptions) Complete(cmd *cobra.Command) error {
	err := o.scribeOptions.Complete()
	if err != nil {
		return err
	}
	o.DestNamespace = o.scribeOptions.destNamespace
	if len(o.DestName) == 0 {
		o.DestName = o.DestNamespace + "-destination"
	}
	klog.V(2).Infof("replication destination %s will be created in %s namespace", o.DestName, o.DestNamespace)
	return nil
}

// Validate validates ReplicationDestination options.
func (o *destinationOptions) Validate() error {
	if len(o.DestCopyMethod) == 0 {
		return fmt.Errorf("must provide --copy-method; one of 'None|Clone|Snapshot'")
	}
	if len(o.DestCapacity) == 0 && len(o.DestPVC) == 0 {
		return fmt.Errorf("must either provide --dest-capacity & --dest-access-mode OR --dest-pvc")
	}
	if len(o.DestAccessMode) == 0 && len(o.DestPVC) == 0 {
		return fmt.Errorf("must either provide --dest-capacity & --dest-access-mode OR --dest-pvc")
	}
	return nil
}

// CreateReplicationDestination creates a ReplicationDestination resource
func (o *destinationOptions) CreateReplicationDestination() error {
	c := &commonOptions{}
	switch {
	case len(o.DestCapacity) > 0:
		capacity := resource.MustParse(o.DestCapacity)
		c.capacity = &capacity
	default:
		c.capacity = nil
	}
	if o.Port == 0 {
		c.port = nil
	}
	switch o.DestCopyMethod {
	case "None", "none":
		c.copyMethod = scribev1alpha1.CopyMethodNone
	case "Clone", "clone":
		c.copyMethod = scribev1alpha1.CopyMethodClone
	case "Snapshot", "snapshot", "SnapShot":
		c.copyMethod = scribev1alpha1.CopyMethodSnapshot
	default:
		return fmt.Errorf("unrecognized --dest-copy-method: %s", o.DestCopyMethod)
	}
	if len(o.DestAccessMode) > 0 {
		switch o.DestAccessMode {
		case "ReadWriteOnce":
			c.accessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
		case "ReadWriteMany":
			c.accessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}
		case "ReadOnlyMany":
			c.accessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadOnlyMany}
		default:
			return fmt.Errorf("unrecognized --dest-access-mode %s", o.DestAccessMode)
		}
	}
	switch {
	case len(o.DestServiceType) > 0:
		switch o.DestServiceType {
		case "ClusterIP", "clusterip", "clusterIP":
			c.serviceType = corev1.ServiceTypeClusterIP
		case "LoadBalancer", "loadbalancer", "Loadbalancer":
			c.serviceType = corev1.ServiceTypeLoadBalancer
		default:
			return fmt.Errorf("unrecognized --dest-service-type %s", o.DestServiceType)
		}
	// if not set, then default to clusterIP
	default:
		c.serviceType = corev1.ServiceTypeClusterIP
	}

	if len(o.Address) > 0 {
		c.address = &o.Address
	}
	switch {
	case len(o.sshKeysSecretOptions.SSHKeysSecret) > 0:
		c.sshKeysSecret = &o.sshKeysSecretOptions.SSHKeysSecret
	default:
		c.sshKeysSecret = nil
	}
	if len(o.SSHUser) > 0 {
		c.sshUser = &o.SSHUser
	}
	if len(o.Path) > 0 {
		c.path = &o.Path
	}
	if len(o.DestStorageClassName) > 0 {
		c.storageClassName = &o.DestStorageClassName
	}
	if len(o.DestVolumeSnapshotClassName) > 0 {
		c.volumeSnapClassName = &o.DestVolumeSnapshotClassName
	}
	if len(o.DestPVC) > 0 {
		c.pvc = &o.DestPVC
	}
	c.parameters = make(map[string]string)
	if len(o.ProviderParameters) > 0 {
		p := strings.Split(o.ProviderParameters, ",")
		for _, kv := range p {
			pair := strings.Split(kv, "/")
			if len(pair) != 2 {
				return fmt.Errorf("error parsing --provider-parameters %s, must be passed as key/value,key1/value1...", o.ProviderParameters)
			}
			c.parameters[pair[0]] = pair[1]
		}
	}
	triggerSpec := &scribev1alpha1.ReplicationDestinationTriggerSpec{
		Schedule: &o.DestSchedule,
	}
	if len(o.DestSchedule) == 0 {
		triggerSpec = nil
	}
	rsyncSpec := &scribev1alpha1.ReplicationDestinationRsyncSpec{
		ReplicationDestinationVolumeOptions: scribev1alpha1.ReplicationDestinationVolumeOptions{
			CopyMethod:              c.copyMethod,
			Capacity:                c.capacity,
			StorageClassName:        c.storageClassName,
			AccessModes:             c.accessModes,
			VolumeSnapshotClassName: c.volumeSnapClassName,
			DestinationPVC:          c.pvc,
		},
		SSHKeys:     c.sshKeysSecret,
		SSHUser:     c.sshUser,
		Address:     c.address,
		ServiceType: &c.serviceType,
		Port:        c.port,
		Path:        c.path,
	}
	var externalSpec *scribev1alpha1.ReplicationDestinationExternalSpec
	if len(o.Provider) > 0 && c.parameters != nil {
		externalSpec = &scribev1alpha1.ReplicationDestinationExternalSpec{
			Provider:   o.Provider,
			Parameters: c.parameters,
		}
	}
	rd := &scribev1alpha1.ReplicationDestination{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "scribe.backube/v1alpha1",
			Kind:       "ReplicationDestination",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      o.DestName,
			Namespace: o.DestNamespace,
		},
		Spec: scribev1alpha1.ReplicationDestinationSpec{
			Trigger:  triggerSpec,
			Rsync:    rsyncSpec,
			External: externalSpec,
		},
	}
	if err := o.scribeOptions.DestinationClient.Create(context.TODO(), rd); err != nil {
		return err
	}
	klog.V(0).Infof("ReplicationDestination %s created in namespace %s", o.DestName, o.DestNamespace)
	return nil
}
