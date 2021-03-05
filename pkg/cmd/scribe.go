package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	kcmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/templates"
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
)

type scribeOptions struct {
	destKubeContext       string
	sourceKubeContext     string
	destKubeClusterName   string
	sourceKubeClusterName string

	genericclioptions.IOStreams
}

func (o *scribeOptions) Bind(flags *pflag.FlagSet) {
	flags.StringVar(&o.destKubeContext, "dest-kube-context", o.destKubeContext, "the name of the kubeconfig context to use for the destination cluster. Defaults to current-context.")
	flags.StringVar(&o.sourceKubeContext, "source-kube-context", o.sourceKubeContext, "the name of the kubeconfig context to use for the destination cluster. Defaults to current-context.")
	flags.StringVar(&o.destKubeClusterName, "dest-kube-clustername", o.destKubeClusterName, "the name of the kubeconfig cluster to use for the destination cluster. Defaults to current-cluster.")
	flags.StringVar(&o.sourceKubeClusterName, "source-kube-clustername", o.sourceKubeClusterName, "the name of the kubeconfig cluster to use for the destination cluster. Defaults to current cluster.")
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

	return cmds
}
