package cmd

import (
	"context"
	"fmt"

	scribev1alpha1 "github.com/backube/scribe/api/v1alpha1"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"
	kcmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/i18n"
	"k8s.io/kubectl/pkg/util/templates"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	scribeSyncSSHSecretLong = templates.LongDesc(`
Scribe is a command line tool for a scribe operator running in a Kubernetes cluster. Scribe asynchronously replicates Kubernetes persistent volumes between clusters or namespaces using rsync, rclone, or restic. Scribe uses a ReplicationDestination and a ReplicationSource to replicate a volume. Data will be synced according to the configured sync schedule.
`)
	scribeSyncSSHSecretExample = templates.Examples(`
	# Copy the SSH secret from the ReplicationDestination namespace to the ReplicationSource namespace.
	# Secret will be copied from namespace 'dest' to namespace 'source'.
    scribe sync-ssh-secret --dest-namespace=dest --source-namespace=source

	# Copy the SSH secret from the ReplicationDestination namespace in one cluster 
	# to the ReplicationSource namespace in another clutser.
	# Secret will be copied from clustername 'kind-kind' context 'kind-kind' namespace 'dest'
	# to context 'admin' clustername 'api-test-com:6443' namespace 'source'.
    scribe sync-ssh-secret --dest-namespace=dest --source-namespace=source \
			--dest-kube-context=kind-kind --dest-clustername=kind-kind \
			--source-kube-context=admin --source-clustername=api-test-com:6443
    `)
)

type sshKeysSecretOptions struct {
	scribeOptions scribeOptions
	SSHKeysSecret string

	genericclioptions.IOStreams
}

func NewCmdScribeSyncSSHSecret(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewSSHKeysSecretOptions(streams)
	cmd := &cobra.Command{
		Use:     "sync-ssh-secret [OPTIONS]",
		Short:   i18n.T("Copy the SSH secret for rsync between namespaces and/or clusters."),
		Long:    fmt.Sprintf(scribeSyncSSHSecretLong),
		Example: fmt.Sprintf(scribeSyncSSHSecretExample),
		Run: func(cmd *cobra.Command, args []string) {
			kcmdutil.CheckErr(o.Complete())
			//TODO: kcmdutil.CheckErr(o.Validate())
			kcmdutil.CheckErr(o.SyncSSHSecret())
		},
	}
	flags := cmd.Flags()
	o.Bind(flags)
	o.scribeOptions.Bind(flags)

	return cmd
}
func (o *sshKeysSecretOptions) Bind(flags *pflag.FlagSet) {
	flags.StringVar(&o.SSHKeysSecret, "ssh-keys-secret", o.SSHKeysSecret, "name of an existing valid SSHKeys secret to be used for authentication. If not set, the default SSHKey secret-name will be used from the ReplicationDestination location (default '<scribe-rsync->dest-src-<name-of-replication-destination>)'.")
}

func NewSSHKeysSecretOptions(streams genericclioptions.IOStreams) *sshKeysSecretOptions {
	return &sshKeysSecretOptions{
		IOStreams: streams,
	}
}

// Complete takes the cmd and infers options.
func (o *sshKeysSecretOptions) Complete() error {
	ctx := context.Background()
	err := o.scribeOptions.Complete()
	if err != nil {
		return err
	}
	repDests := &scribev1alpha1.ReplicationDestinationList{}
	opts := []client.ListOption{
		client.InNamespace(o.scribeOptions.destNamespace),
	}

	err = o.scribeOptions.DestinationClient.List(ctx, repDests, opts...)
	if err != nil {
		return err
	}

	if len(o.SSHKeysSecret) == 0 {
		o.SSHKeysSecret = "scribe-rsync-dest-src-" + repDests.Items[0].Name
	}
	return nil
}

func (o *sshKeysSecretOptions) SyncSSHSecret() error {
	ctx := context.Background()
	originalSecret := &corev1.Secret{}
	nsName := types.NamespacedName{
		Namespace: o.scribeOptions.destNamespace,
		Name:      o.SSHKeysSecret,
	}
	err := o.scribeOptions.DestinationClient.Get(ctx, nsName, originalSecret)
	if err != nil {
		return err
	}
	newSecret := originalSecret.DeepCopy()
	newSecret.ObjectMeta = metav1.ObjectMeta{
		Name:            originalSecret.ObjectMeta.Name,
		Namespace:       o.scribeOptions.sourceNamespace,
		OwnerReferences: nil,
	}

	err = o.scribeOptions.SourceClient.Create(ctx, newSecret)
	if err != nil {
		return err
	}
	klog.Infof("secret %s created in namespace %s", o.SSHKeysSecret, o.scribeOptions.sourceNamespace)
	return nil
}
