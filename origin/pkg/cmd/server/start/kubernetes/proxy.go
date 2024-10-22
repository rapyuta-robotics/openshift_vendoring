package kubernetes

import (
	"fmt"
	"io"
	"os"

	"github.com/openshift/github.com/spf13/cobra"

	proxyapp "github.com/openshift/kubernetes/cmd/kube-proxy/app"
	proxyoptions "github.com/openshift/kubernetes/cmd/kube-proxy/app/options"
	kcmdutil "github.com/openshift/kubernetes/pkg/kubectl/cmd/util"
	kflag "github.com/openshift/kubernetes/pkg/util/flag"
	"github.com/openshift/kubernetes/pkg/util/logs"
)

const proxyLong = `
Start Kubernetes Proxy

This command launches an instance of the Kubernetes proxy (kube-proxy).`

// NewProxyCommand provides a CLI handler for the 'proxy' command
func NewProxyCommand(name, fullName string, out io.Writer) *cobra.Command {
	proxyConfig := proxyoptions.NewProxyConfig()

	cmd := &cobra.Command{
		Use:   name,
		Short: "Launch Kubernetes proxy (kube-proxy)",
		Long:  proxyLong,
		Run: func(c *cobra.Command, args []string) {
			startProfiler()

			logs.InitLogs()
			defer logs.FlushLogs()

			s, err := proxyapp.NewProxyServerDefault(proxyConfig)
			kcmdutil.CheckErr(err)

			if err := s.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
		},
	}
	cmd.SetOutput(out)

	flags := cmd.Flags()
	flags.SetNormalizeFunc(kflag.WordSepNormalizeFunc)
	proxyConfig.AddFlags(flags)

	return cmd
}
