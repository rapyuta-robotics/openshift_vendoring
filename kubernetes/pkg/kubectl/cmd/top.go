/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"io"

	cmdutil "github.com/openshift/kubernetes/pkg/kubectl/cmd/util"

	"github.com/openshift/github.com/spf13/cobra"
	"github.com/openshift/kubernetes/pkg/kubectl/cmd/templates"
)

// TopOptions contains all the options for running the top cli command.
type TopOptions struct{}

var (
	topLong = templates.LongDesc(`
		Display Resource (CPU/Memory/Storage) usage.

		The top command allows you to see the resource consumption for nodes or pods.`)
)

func NewCmdTop(f cmdutil.Factory, out, errOut io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "top",
		Short: "Display Resource (CPU/Memory/Storage) usage",
		Long:  topLong,
		Run:   cmdutil.DefaultSubCommandRun(errOut),
	}

	// create subcommands
	cmd.AddCommand(NewCmdTopNode(f, out))
	cmd.AddCommand(NewCmdTopPod(f, out))
	return cmd
}
