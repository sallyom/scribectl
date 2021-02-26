package cmd

import (
	"flag"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	kcmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/templates"
)

var (
	scribeLong = templates.LongDesc(`
        Scribe is a command line tool for a scribe operator running in a Kubernetes cluster.
        Scribe asynchronously replicates Kubernetes persistent volumes between clusters or namespaces
		using rsync, rclone, or restic. Scribe uses a ReplicationDestination and a ReplicationSource
		to replicate a volume. Data will be synced according to the configured sync schedule.
	`)
	scribeExplain = templates.LongDesc(`
    To start using Scribe, login to your cluster and install the Scribe operator.
	Installation instructions at https://scribe-replication.readthedocs.io/en/latest/installation/index.html

    For more on Scribe, see the documentation at https://scribe-replication.readthedocs.io/

    To see the full list of commands supported, run 'scribe --help'.`)
)

// NewCmdScribe implements the scribe command
func NewCmdScribe(in io.Reader, out, errout io.Writer) *cobra.Command {
	// main command
	streams := genericclioptions.IOStreams{In: in, Out: out, ErrOut: errout}
	cmds := &cobra.Command{
		Use:   "scribe",
		Short: "Asynchronously replicate persistent volumes.",
		Long:  scribeLong,
		Run: func(c *cobra.Command, args []string) {
			c.SetOutput(errout)
			kcmdutil.RequireNoArguments(c, args)
			fmt.Fprintf(errout, "%s\n\n%s\n", scribeLong, scribeExplain)
		},
	}
	kubeConfigFlags := genericclioptions.NewConfigFlags(true)
	matchVersionKubeConfigFlags := kcmdutil.NewMatchVersionFlags(kubeConfigFlags)
	matchVersionKubeConfigFlags.AddFlags(cmds.PersistentFlags())
	cmds.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	f := kcmdutil.NewFactory(matchVersionKubeConfigFlags)

	cmds.AddCommand(NewCmdScribeNewDestination(f, streams))
	cmds.AddCommand(NewCmdScribeNewSource(f, streams))

	return cmds
}
