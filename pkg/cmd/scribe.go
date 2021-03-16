package cmd

import (
	"fmt"
	"io"

	scribev1alpha1 "github.com/backube/scribe/api/v1alpha1"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	kcmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/templates"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	scribeLong = templates.LongDesc(`
Scribe is a command line tool for a scribe operator running in a Kubernetes cluster. Scribe asynchronously replicates Kubernetes persistent volumes between clusters or namespaces using rsync, rclone, or restic. Scribe uses a ReplicationDestination and a ReplicationSourc to replicate a volume. Data will be synced according to the configured sync schedule.
`)
	scribeExplain = templates.LongDesc(`
    To start using Scribe, login to your cluster and install the Scribe operator.
	Installation instructions at https://scribe-replication.readthedocs.io/en/latest/installation/index.html

    For more on Scribe, see the documentation at https://scribe-replication.readthedocs.io/

    To see the full list of commands supported, run 'scribe --help'.`)

	scribeConfig = "scribe-config"
)

type scribeOptions struct {
	destKubeContext       string
	sourceKubeContext     string
	destKubeClusterName   string
	sourceKubeClusterName string
	destNamespace         string
	sourceNamespace       string
	DestinationClient     client.Client
	SourceClient          client.Client

	genericclioptions.IOStreams
}

func (o *scribeOptions) bindFlags(cmd *cobra.Command, v *viper.Viper) {
	flags := cmd.Flags()
	flags.StringVar(&o.destKubeContext, "dest-kube-context", o.destKubeContext, "the name of the kubeconfig context to use for the destination cluster. Defaults to current-context.")
	flags.StringVar(&o.sourceKubeContext, "source-kube-context", o.sourceKubeContext, "the name of the kubeconfig context to use for the destination cluster. Defaults to current-context.")
	flags.StringVar(&o.destKubeClusterName, "dest-kube-clustername", o.destKubeClusterName, "the name of the kubeconfig cluster to use for the destination cluster. Defaults to current-cluster.")
	flags.StringVar(&o.sourceKubeClusterName, "source-kube-clustername", o.sourceKubeClusterName, "the name of the kubeconfig cluster to use for the destination cluster. Defaults to current cluster.")
	flags.StringVar(&o.destNamespace, "dest-namespace", o.destNamespace, "the transfer destination namespace and/or location of a ReplicationDestination. This namespace must exist. If not set, use the current namespace.")
	flags.StringVar(&o.sourceNamespace, "source-namespace", o.sourceNamespace, "the transfer source namespace and/or location of a ReplicationSource. This namespace must exist. If not set, use the current namespace.")
	flags.VisitAll(func(f *pflag.Flag) {
		// Apply the viper config value to the flag when the flag is not set and viper has a value
		if v.IsSet(f.Name) {
			val := v.Get(f.Name)
			flags.Set(f.Name, fmt.Sprintf("%v", val))
		}
	})
}

func (o *scribeOptions) Bind(cmd *cobra.Command, v *viper.Viper) error {
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

// NewCmdScribe implements the scribe command
func NewCmdScribe(in io.Reader, out, errout io.Writer) *cobra.Command {
	// main command
	streams := genericclioptions.IOStreams{In: in, Out: out, ErrOut: errout}
	cmds := &cobra.Command{
		Use:   "scribe",
		Short: "Asynchronously replicate persistent volumes.",
		Long:  fmt.Sprintf(scribeLong),
		//Run: kcmdutil.DefaultSubCommandRun(streams.ErrOut),
		Run: func(c *cobra.Command, args []string) {
			c.SetOutput(errout)
			kcmdutil.RequireNoArguments(c, args)
			fmt.Fprintf(errout, "%s\n\n%s\n", scribeLong, scribeExplain)
		},
	}
	// TODO: Maybe pass --dest-kube-context and --source-kube-context and get 2 factories?
	// For switching contexts: https://github.com/kubernetes/client-go/issues/192#issuecomment-362775792
	cmds.AddCommand(NewCmdScribeNewDestination(streams))
	cmds.AddCommand(NewCmdScribeNewSource(streams))
	cmds.AddCommand(NewCmdScribeSyncSSHSecret(streams))

	return cmds
}

func (o *scribeOptions) Complete() error {
	destKubeConfigFlags := genericclioptions.NewConfigFlags(true)
	if len(o.destKubeContext) > 0 {
		destKubeConfigFlags.Context = &o.destKubeContext
	}
	if len(o.destKubeClusterName) > 0 {
		destKubeConfigFlags.ClusterName = &o.destKubeClusterName
	}
	sourceKubeConfigFlags := genericclioptions.NewConfigFlags(true)
	if len(o.sourceKubeContext) > 0 {
		sourceKubeConfigFlags.Context = &o.sourceKubeContext
	}
	if len(o.sourceKubeClusterName) > 0 {
		sourceKubeConfigFlags.ClusterName = &o.sourceKubeClusterName
	}
	destf := kcmdutil.NewFactory(destKubeConfigFlags)
	sourcef := kcmdutil.NewFactory(sourceKubeConfigFlags)

	// get client and namespace
	destClientConfig, err := destf.ToRESTConfig()
	if err != nil {
		return err
	}
	sourceClientConfig, err := sourcef.ToRESTConfig()
	if err != nil {
		return err
	}
	scheme := runtime.NewScheme()
	scribev1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)
	destKClient, err := client.New(destClientConfig, client.Options{Scheme: scheme})
	if err != nil {
		return err
	}
	o.DestinationClient = destKClient
	sourceKClient, err := client.New(sourceClientConfig, client.Options{Scheme: scheme})
	if err != nil {
		return err
	}
	o.SourceClient = sourceKClient
	if len(o.destNamespace) == 0 {
		o.destNamespace, _, err = destf.ToRawKubeConfigLoader().Namespace()
		if err != nil {
			return err
		}
	}
	if len(o.sourceNamespace) == 0 {
		o.sourceNamespace, _, err = sourcef.ToRawKubeConfigLoader().Namespace()
		if err != nil {
			return err
		}
	}
	return nil
}
