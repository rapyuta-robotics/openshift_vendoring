package importer

import (
	"fmt"
	"io"

	"github.com/openshift/github.com/spf13/cobra"
	cmdutil "github.com/openshift/kubernetes/pkg/kubectl/cmd/util"

	"github.com/openshift/origin/pkg/cmd/templates"
	"github.com/openshift/origin/pkg/cmd/util/clientcmd"
)

var (
	importLong = templates.LongDesc(`
		Import outside applications into OpenShift

		These commands assist in bringing existing applications into OpenShift.`)
)

// NewCmdImport exposes commands for modifying objects.
func NewCmdImport(fullName string, f *clientcmd.Factory, in io.Reader, out, errout io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import COMMAND",
		Short: "Commands that import applications",
		Long:  importLong,
		Run:   cmdutil.DefaultSubCommandRun(errout),
	}

	name := fmt.Sprintf("%s import", fullName)

	cmd.AddCommand(NewCmdDockerCompose(name, f, in, out, errout))
	cmd.AddCommand(NewCmdAppJSON(name, f, in, out, errout))
	return cmd
}
