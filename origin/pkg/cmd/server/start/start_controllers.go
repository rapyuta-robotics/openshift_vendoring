package start

import (
	"fmt"
	"io"
	"os"

	"github.com/golang/glog"
	"github.com/openshift/github.com/spf13/cobra"

	kerrors "github.com/openshift/kubernetes/pkg/api/errors"
	kcmdutil "github.com/openshift/kubernetes/pkg/kubectl/cmd/util"

	"github.com/openshift/origin/pkg/cmd/flagtypes"
	configapi "github.com/openshift/origin/pkg/cmd/server/api"
	"github.com/openshift/origin/pkg/cmd/templates"
)

var controllersLong = templates.LongDesc(`
	Start the master controllers

	This command starts the controllers for the master.  Running

	    %[1]s start master %[2]s

	will start the controllers that manage the master state, including the scheduler. The controllers
	will run in the foreground until you terminate the process.`)

// NewCommandStartMasterControllers starts only the controllers
func NewCommandStartMasterControllers(name, basename string, out, errout io.Writer) (*cobra.Command, *MasterOptions) {
	options := &MasterOptions{Output: out}
	options.DefaultsFromName(basename)

	cmd := &cobra.Command{
		Use:   "controllers",
		Short: "Launch master controllers",
		Long:  fmt.Sprintf(controllersLong, basename, name),
		Run: func(c *cobra.Command, args []string) {
			if err := options.Complete(); err != nil {
				fmt.Fprintln(errout, kcmdutil.UsageError(c, err.Error()))
				return
			}

			if len(options.ConfigFile) == 0 {
				fmt.Fprintln(errout, kcmdutil.UsageError(c, "--config is required for this command"))
				return
			}

			if err := options.Validate(args); err != nil {
				fmt.Fprintln(errout, kcmdutil.UsageError(c, err.Error()))
				return
			}

			startProfiler()

			if err := options.StartMaster(); err != nil {
				if kerrors.IsInvalid(err) {
					if details := err.(*kerrors.StatusError).ErrStatus.Details; details != nil {
						fmt.Fprintf(errout, "Invalid %s %s\n", details.Kind, details.Name)
						for _, cause := range details.Causes {
							fmt.Fprintf(errout, "  %s: %s\n", cause.Field, cause.Message)
						}
						os.Exit(255)
					}
				}
				glog.Fatal(err)
			}
		},
	}

	// start controllers on a non conflicting health port from the default master
	listenArg := &ListenArg{
		ListenAddr: flagtypes.Addr{
			Value:         "127.0.0.1:8444",
			DefaultScheme: "https",
			DefaultPort:   8444,
			AllowPrefix:   true,
		}.Default(),
	}

	options.MasterArgs = NewDefaultMasterArgs()
	options.MasterArgs.StartControllers = true
	options.MasterArgs.OverrideConfig = func(config *configapi.MasterConfig) error {
		config.ServingInfo.BindAddress = listenArg.ListenAddr.URL.Host
		return nil
	}

	flags := cmd.Flags()
	// This command only supports reading from config and the listen argument
	flags.StringVar(&options.ConfigFile, "config", "", "Location of the master configuration file to run from. Required")
	cmd.MarkFlagFilename("config", "yaml", "yml")
	BindListenArg(listenArg, flags, "")

	return cmd, options
}
