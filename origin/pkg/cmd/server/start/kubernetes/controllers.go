package kubernetes

import (
	"fmt"
	"io"
	"os"

	"github.com/openshift/github.com/spf13/cobra"

	controllerapp "github.com/openshift/kubernetes/cmd/kube-controller-manager/app"
	controlleroptions "github.com/openshift/kubernetes/cmd/kube-controller-manager/app/options"
	kflag "github.com/openshift/kubernetes/pkg/util/flag"
	"github.com/openshift/kubernetes/pkg/util/logs"
)

const controllersLong = `
Start Kubernetes controller manager

This command launches an instance of the Kubernetes controller-manager (kube-controller-manager).`

// NewControllersCommand provides a CLI handler for the 'controller-manager' command
func NewControllersCommand(name, fullName string, out io.Writer) *cobra.Command {
	controllerOptions := controlleroptions.NewCMServer()

	cmd := &cobra.Command{
		Use:   name,
		Short: "Launch Kubernetes controller manager (kube-controller-manager)",
		Long:  controllersLong,
		Run: func(c *cobra.Command, args []string) {
			startProfiler()

			logs.InitLogs()
			defer logs.FlushLogs()

			if err := controllerapp.Run(controllerOptions); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
		},
	}
	cmd.SetOutput(out)

	flags := cmd.Flags()
	flags.SetNormalizeFunc(kflag.WordSepNormalizeFunc)
	controllerOptions.AddFlags(flags)

	return cmd
}
