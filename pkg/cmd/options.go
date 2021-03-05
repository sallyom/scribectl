package cmd

import (
	"context"
	"fmt"
	"strings"

	scribev1alpha1 "github.com/backube/scribe/api/v1alpha1"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"
)

type ReplicationOptions struct {
	scribeOptions           scribeOptions
	Mode                    string
	CopyMethod              string //v1alpha1.CopyMethodType
	Capacity                string //*resource.Quantity
	StorageClassName        string
	AccessMode              string //[]corev1.PersistentVolumeAccessMode
	Address                 string
	VolumeSnapshotClassName string
	PVC                     string
	Schedule                string
	SSHKeys                 string
	SSHUser                 string
	Name                    string
	Namespace               string
	ServiceType             string //*corev1.ServiceType
	Port                    int32  //int32
	Path                    string
	RcloneConfig            string
	Provider                string
	ProviderParameters      string //map[string]string
	genericclioptions.IOStreams
}

type commonOptions struct {
	capacity            *resource.Quantity
	copyMethod          scribev1alpha1.CopyMethodType
	accessModes         []corev1.PersistentVolumeAccessMode
	serviceType         corev1.ServiceType
	externalSpec        *scribev1alpha1.ReplicationDestinationExternalSpec
	address             *string
	sshKeys             *string
	sshUser             *string
	path                *string
	storageClassName    *string
	volumeSnapClassName *string
	pvc                 *string
	port                *int32
	parameters          map[string]string
}

func NewReplicationOptions(streams genericclioptions.IOStreams) *ReplicationOptions {
	return &ReplicationOptions{
		IOStreams: streams,
	}
}

func (o *ReplicationOptions) Complete(cmd *cobra.Command) error {
	err := o.scribeOptions.Complete()
	if err != nil {
		return err
	}
	switch o.Mode {
	case "destination":
		o.Namespace = o.scribeOptions.destNamespace
	case "source":
		o.Namespace = o.scribeOptions.sourceNamespace
	}
	if len(o.Name) == 0 {
		o.Name = o.Namespace + "-scribe-" + o.Mode
	}
	klog.V(2).Infof("replication %s %s will be created in %s namespace", o.Mode, o.Name, o.Namespace)
	return nil
}

// Validate validates ReplicationDestination options.
func (o *ReplicationOptions) Validate() error {
	if len(o.CopyMethod) == 0 {
		return fmt.Errorf("must provide --copy-method; one of 'None|Clone|Snapshot'")
	}
	if o.Mode == "source" {
		return nil
	}
	if len(o.Capacity) == 0 && len(o.PVC) == 0 {
		return fmt.Errorf("must either provide --dest-capacity & --dest-access-mode OR --dest-pvc")
	}
	if len(o.AccessMode) == 0 && len(o.PVC) == 0 {
		return fmt.Errorf("must either provide --dest-capacity & --dest-access-mode OR --dest-pvc")
	}
	return nil
}

func (o *ReplicationOptions) getCommonOptions() (*commonOptions, error) {
	c := &commonOptions{}
	switch {
	case len(o.Capacity) > 0:
		capacity := resource.MustParse(o.Capacity)
		c.capacity = &capacity
	default:
		c.capacity = nil
	}
	if o.Port == 0 {
		c.port = nil
	}
	switch o.CopyMethod {
	case "None", "none":
		c.copyMethod = scribev1alpha1.CopyMethodNone
	case "Clone", "clone":
		c.copyMethod = scribev1alpha1.CopyMethodClone
	case "Snapshot", "snapshot", "SnapShot":
		c.copyMethod = scribev1alpha1.CopyMethodSnapshot
	default:
		return nil, fmt.Errorf("unrecognized --dest-copy-method: %s", o.CopyMethod)
	}
	if len(o.AccessMode) > 0 {
		switch o.AccessMode {
		case "ReadWriteOnce":
			c.accessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
		case "ReadWriteMany":
			c.accessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}
		case "ReadOnlyMany":
			c.accessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadOnlyMany}
		default:
			return nil, fmt.Errorf("unrecognized --dest-access-mode %s", o.AccessMode)
		}
	}
	switch {
	case len(o.ServiceType) > 0:
		switch o.ServiceType {
		case "ClusterIP", "clusterip", "clusterIP":
			c.serviceType = corev1.ServiceTypeClusterIP
		case "LoadBalancer", "loadbalancer", "Loadbalancer":
			c.serviceType = corev1.ServiceTypeLoadBalancer
		default:
			return nil, fmt.Errorf("unrecognized --dest-service-type %s", o.ServiceType)
		}
	// if not set, then default to clusterIP
	default:
		c.serviceType = corev1.ServiceTypeClusterIP
	}

	if len(o.Address) > 0 {
		c.address = &o.Address
	}
	if len(o.SSHKeys) > 0 {
		c.sshKeys = &o.SSHKeys
	}
	if len(o.SSHUser) > 0 {
		c.sshUser = &o.SSHUser
	}
	if len(o.Path) > 0 {
		c.path = &o.Path
	}
	if len(o.StorageClassName) > 0 {
		c.storageClassName = &o.StorageClassName
	}
	if len(o.VolumeSnapshotClassName) > 0 {
		c.volumeSnapClassName = &o.VolumeSnapshotClassName
	}
	if len(o.PVC) > 0 {
		c.pvc = &o.PVC
	}
	c.parameters = make(map[string]string)
	if len(o.ProviderParameters) > 0 {
		p := strings.Split(o.ProviderParameters, ",")
		for _, kv := range p {
			pair := strings.Split(kv, "/")
			if len(pair) != 2 {
				return nil, fmt.Errorf("error parsing --provider-parameters %s, must be passed as key/value,key1/value1...", o.ProviderParameters)
			}
			c.parameters[pair[0]] = pair[1]
		}
	}
	return c, nil
}

// CreateReplicationDestination creates a ReplicationDestination resource
func (o *ReplicationOptions) CreateReplicationDestination() error {
	c, err := o.getCommonOptions()
	if err != nil {
		return err
	}
	triggerSpec := &scribev1alpha1.ReplicationDestinationTriggerSpec{
		Schedule: &o.Schedule,
	}
	if len(o.Schedule) == 0 {
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
		SSHKeys:     c.sshKeys,
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
			Name:      o.Name,
			Namespace: o.Namespace,
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
	klog.V(0).Infof("ReplicationDestination %s created in namespace %s", o.Name, o.Namespace)
	return nil
}

// CreateReplicationSource creates a ReplicationSource resource
func (o *ReplicationOptions) CreateReplicationSource() error {
	c, err := o.getCommonOptions()
	if err != nil {
		return err
	}
	triggerSpec := &scribev1alpha1.ReplicationSourceTriggerSpec{
		Schedule: &o.Schedule,
	}
	if len(o.Schedule) == 0 {
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
		SSHKeys:     c.sshKeys,
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
			Name:      o.Name,
			Namespace: o.Namespace,
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
	klog.V(0).Infof("ReplicationSource %s created in namespace %s", o.Name, o.Namespace)
	return nil
}
