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

type sourceOptions struct {
	scribeOptions                 scribeOptions
	sshKeysSecretOptions          sshKeysSecretOptions
	SourceCopyMethod              string //v1alpha1.CopyMethodType
	SourceCapacity                string //*resource.Quantity
	SourceStorageClassName        string
	SourceAccessMode              string //[]corev1.PersistentVolumeAccessMode
	Address                       string
	SourceVolumeSnapshotClassName string
	SourcePVC                     string
	SourceSchedule                string
	SSHUser                       string
	SourceName                    string
	SourceNamespace               string
	SourceServiceType             string //*corev1.ServiceType
	Port                          int32  //int32
	Path                          string
	RcloneConfig                  string
	Provider                      string
	ProviderParameters            string //map[string]string
	genericclioptions.IOStreams
}

func NewCmdScribeNewSource(streams genericclioptions.IOStreams) *cobra.Command {
	v := viper.New()
	o := NewSourceOptions(streams)
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
	kcmdutil.CheckErr(o.scribeOptions.Bind(cmd, v))
	kcmdutil.CheckErr(o.sshKeysSecretOptions.Bind(cmd, v))
	kcmdutil.CheckErr(o.Bind(cmd, v))

	return cmd
}

func NewSourceOptions(streams genericclioptions.IOStreams) *sourceOptions {
	return &sourceOptions{
		IOStreams: streams,
	}
}

func (o *sourceOptions) bindFlags(cmd *cobra.Command, v *viper.Viper) error {
	flags := cmd.Flags()
	flags.StringVar(&o.SourceCopyMethod, "source-copy-method", o.SourceCopyMethod, "the method of creating a point-in-time image of the source volume; one of 'None|Clone|Snapshot'")
	flags.StringVar(&o.Address, "address", o.Address, "the remote address to connect to for replication.")
	flags.StringVar(&o.SourceCapacity, "source-capacity", o.SourceCapacity, "provided to override the capacity of the point-in-Time image.")
	flags.StringVar(&o.SourceStorageClassName, "source-storage-class-name", o.SourceStorageClassName, "provided to override the StorageClass of the point-in-Time image.")
	flags.StringVar(&o.SourceAccessMode, "source-access-mode", o.SourceAccessMode, "provided to override the accessModes of the point-in-Time image. One of 'ReadWriteOnce|ReadOnlyMany|ReadWriteMany")
	flags.StringVar(&o.SourceVolumeSnapshotClassName, "source-volume-snapshot-class", o.SourceVolumeSnapshotClassName, "name of the VolumeSnapshotClass to be used for the source volume, only if the copyMethod is 'Snapshot'. If not set, the default VSC will be used.")
	flags.StringVar(&o.SourcePVC, "source-pvc", o.SourcePVC, "name of an existing PersistentVolumeClaim (PVC) to replicate.")
	// TODO: Default to every 3min for source?
	flags.StringVar(&o.SourceSchedule, "source-cron-spec", "*/3 * * * *", "cronspec to be used to schedule capturing the state of the source volume. If not set the source volume will be captured every 3 minutes.")
	// Defaults to "root" after creation
	flags.StringVar(&o.SSHUser, "source-ssh-user", o.SSHUser, "username for outgoing SSH connections (default 'root')")
	// Defaults to ClusterIP after creation
	flags.StringVar(&o.SourceServiceType, "source-service-type", o.SourceServiceType, "one of ClusterIP|LoadBalancer. Service type that will be created for incoming SSH connections. (default 'ClusterIP')")
	// TODO: Defaulted in CLI, should it be??
	flags.StringVar(&o.SourceName, "source-name", o.SourceName, "name of the ReplicationSource resource (default '<source-ns>-scribe-source')")
	// defaults to 22 after creation
	flags.Int32Var(&o.Port, "port", o.Port, "SSH port to connect to for replication. (default 22)")
	flags.StringVar(&o.Provider, "provider", o.Provider, "name of an external replication provider, if applicable; pass as 'domain.com/provider'")
	// TODO: I don't know how many params providers have? If a lot, can pass a file instead
	flags.StringVar(&o.ProviderParameters, "provider-parameters", o.ProviderParameters, "provider-specific key/value configuration parameters, if using an external provider; pass as 'key/value,key1/value1,key2/value2'")
	// defaults to "/" after creation
	flags.StringVar(&o.Path, "path", o.Path, "the remote path to rsync to (default '/')")
	cmd.MarkFlagRequired("source-copy-method")
	cmd.MarkFlagRequired("source-pvc")
	flags.VisitAll(func(f *pflag.Flag) {
		if !f.Changed && v.IsSet(f.Name) {
			val := v.Get(f.Name)
			flags.Set(f.Name, fmt.Sprintf("%v", val))
		}
	})
	return nil
}

func (o *sourceOptions) Bind(cmd *cobra.Command, v *viper.Viper) error {
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

func (o *sourceOptions) Complete(cmd *cobra.Command) error {
	err := o.scribeOptions.Complete()
	if err != nil {
		return err
	}
	o.SourceNamespace = o.scribeOptions.sourceNamespace
	if len(o.SourceName) == 0 {
		o.SourceName = o.SourceNamespace + "-source"
	}
	klog.V(2).Infof("replication source %s will be created in %s namespace", o.SourceName, o.SourceNamespace)
	return nil
}

// Validate validates ReplicationSource options.
func (o *sourceOptions) Validate() error {
	if len(o.SourceCopyMethod) == 0 {
		return fmt.Errorf("must provide --copy-method; one of 'None|Clone|Snapshot'")
	}
	//TODO: FIX THIS
	if len(o.sshKeysSecretOptions.SSHKeysSecret) == 0 {
		return fmt.Errorf("must provide the name of the secret in ReplicationSource namespace that holds the SSHKeys for connecting to the ReplicationDestination namespace")
	}
	return nil
}

// CreateReplicationSource creates a ReplicationSource resource
func (o *sourceOptions) CreateReplicationSource() error {
	c := &commonOptions{}
	switch {
	case len(o.SourceCapacity) > 0:
		capacity := resource.MustParse(o.SourceCapacity)
		c.capacity = &capacity
	default:
		c.capacity = nil
	}
	if o.Port == 0 {
		c.port = nil
	}
	switch o.SourceCopyMethod {
	case "None", "none":
		c.copyMethod = scribev1alpha1.CopyMethodNone
	case "Clone", "clone":
		c.copyMethod = scribev1alpha1.CopyMethodClone
	case "Snapshot", "snapshot", "SnapShot":
		c.copyMethod = scribev1alpha1.CopyMethodSnapshot
	default:
		return fmt.Errorf("unrecognized --dest-copy-method: %s", o.SourceCopyMethod)
	}
	if len(o.SourceAccessMode) > 0 {
		switch o.SourceAccessMode {
		case "ReadWriteOnce":
			c.accessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
		case "ReadWriteMany":
			c.accessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}
		case "ReadOnlyMany":
			c.accessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadOnlyMany}
		default:
			return fmt.Errorf("unrecognized --dest-access-mode %s", o.SourceAccessMode)
		}
	}
	switch {
	case len(o.SourceServiceType) > 0:
		switch o.SourceServiceType {
		case "ClusterIP", "clusterip", "clusterIP":
			c.serviceType = corev1.ServiceTypeClusterIP
		case "LoadBalancer", "loadbalancer", "Loadbalancer":
			c.serviceType = corev1.ServiceTypeLoadBalancer
		default:
			return fmt.Errorf("unrecognized --dest-service-type %s", o.SourceServiceType)
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
	if len(o.SourceStorageClassName) > 0 {
		c.storageClassName = &o.SourceStorageClassName
	}
	if len(o.SourceVolumeSnapshotClassName) > 0 {
		c.volumeSnapClassName = &o.SourceVolumeSnapshotClassName
	}
	if len(o.SourcePVC) > 0 {
		c.pvc = &o.SourcePVC
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
	triggerSpec := &scribev1alpha1.ReplicationSourceTriggerSpec{
		Schedule: &o.SourceSchedule,
	}
	if len(o.SourceSchedule) == 0 {
		triggerSpec = nil
	}
	rsyncSpec := &scribev1alpha1.ReplicationSourceRsyncSpec{
		ReplicationSourceVolumeOptions: scribev1alpha1.ReplicationSourceVolumeOptions{
			CopyMethod:              c.copyMethod,
			Capacity:                c.capacity,
			StorageClassName:        c.storageClassName,
			AccessModes:             c.accessModes,
			VolumeSnapshotClassName: c.volumeSnapClassName,
		},
		SSHKeys:     c.sshKeysSecret,
		ServiceType: &c.serviceType,
		Address:     c.address,
		Port:        c.port,
		Path:        c.path,
		SSHUser:     c.sshUser,
	}
	var externalSpec *scribev1alpha1.ReplicationSourceExternalSpec
	if len(o.Provider) > 0 && c.parameters != nil {
		externalSpec = &scribev1alpha1.ReplicationSourceExternalSpec{
			Provider:   o.Provider,
			Parameters: c.parameters,
		}
	}
	rs := &scribev1alpha1.ReplicationSource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "scribe.backube/v1alpha1",
			Kind:       "ReplicationSource",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      o.SourceName,
			Namespace: o.SourceNamespace,
		},
		Spec: scribev1alpha1.ReplicationSourceSpec{
			SourcePVC: *c.pvc,
			Trigger:   triggerSpec,
			Rsync:     rsyncSpec,
			External:  externalSpec,
		},
	}
	if err := o.scribeOptions.SourceClient.Create(context.TODO(), rs); err != nil {
		return err
	}
	klog.V(0).Infof("ReplicationSource %s created in namespace %s", o.SourceName, o.SourceNamespace)
	return nil
}
