/*
Copyright 2014 The Kubernetes Authors.

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
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/golang/glog"
	"github.com/openshift/github.com/spf13/cobra"
	"github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/api/meta"
	"github.com/openshift/kubernetes/pkg/kubectl"
	"github.com/openshift/kubernetes/pkg/kubectl/cmd/templates"
	cmdutil "github.com/openshift/kubernetes/pkg/kubectl/cmd/util"
	"github.com/openshift/kubernetes/pkg/kubectl/resource"
	"github.com/openshift/kubernetes/pkg/runtime"
	utilerrors "github.com/openshift/kubernetes/pkg/util/errors"
	"github.com/openshift/kubernetes/pkg/util/strategicpatch"
	"github.com/openshift/kubernetes/pkg/util/validation"
)

// LabelOptions have the data required to perform the label operation
type LabelOptions struct {
	// Filename options
	resource.FilenameOptions

	// Common user flags
	overwrite       bool
	local           bool
	dryrun          bool
	all             bool
	resourceVersion string
	selector        string
	outputFormat    string

	// results of arg parsing
	resources    []string
	newLabels    map[string]string
	removeLabels []string

	// Common shared fields
	out io.Writer
}

var (
	label_long = templates.LongDesc(`
		Update the labels on a resource.

		* A label must begin with a letter or number, and may contain letters, numbers, hyphens, dots, and underscores, up to %[1]d characters.
		* If --overwrite is true, then existing labels can be overwritten, otherwise attempting to overwrite a label will result in an error.
		* If --resource-version is specified, then updates will use this resource version, otherwise the existing resource-version will be used.`)

	label_example = templates.Examples(`
		# Update pod 'foo' with the label 'unhealthy' and the value 'true'.
		kubectl label pods foo unhealthy=true

		# Update pod 'foo' with the label 'status' and the value 'unhealthy', overwriting any existing value.
		kubectl label --overwrite pods foo status=unhealthy

		# Update all pods in the namespace
		kubectl label pods --all status=unhealthy

		# Update a pod identified by the type and name in "pod.json"
		kubectl label -f pod.json status=unhealthy

		# Update pod 'foo' only if the resource is unchanged from version 1.
		kubectl label pods foo status=unhealthy --resource-version=1

		# Update pod 'foo' by removing a label named 'bar' if it exists.
		# Does not require the --overwrite flag.
		kubectl label pods foo bar-`)
)

func NewCmdLabel(f cmdutil.Factory, out io.Writer) *cobra.Command {
	options := &LabelOptions{}

	// retrieve a list of handled resources from printer as valid args
	validArgs, argAliases := []string{}, []string{}
	p, err := f.Printer(nil, kubectl.PrintOptions{
		ColumnLabels: []string{},
	})
	cmdutil.CheckErr(err)
	if p != nil {
		validArgs = p.HandledResources()
		argAliases = kubectl.ResourceAliases(validArgs)
	}

	cmd := &cobra.Command{
		Use:     "label [--overwrite] (-f FILENAME | TYPE NAME) KEY_1=VAL_1 ... KEY_N=VAL_N [--resource-version=version]",
		Short:   "Update the labels on a resource",
		Long:    fmt.Sprintf(label_long, validation.LabelValueMaxLength),
		Example: label_example,
		Run: func(cmd *cobra.Command, args []string) {
			if err := options.Complete(f, out, cmd, args); err != nil {
				cmdutil.CheckErr(cmdutil.UsageError(cmd, err.Error()))
			}
			if err := options.Validate(); err != nil {
				cmdutil.CheckErr(cmdutil.UsageError(cmd, err.Error()))
			}
			cmdutil.CheckErr(options.RunLabel(f, cmd))
		},
		ValidArgs:  validArgs,
		ArgAliases: argAliases,
	}
	cmdutil.AddPrinterFlags(cmd)
	cmd.Flags().Bool("overwrite", false, "If true, allow labels to be overwritten, otherwise reject label updates that overwrite existing labels.")
	cmd.Flags().Bool("local", false, "If true, label will NOT contact api-server but run locally.")
	cmd.Flags().StringP("selector", "l", "", "Selector (label query) to filter on")
	cmd.Flags().Bool("all", false, "select all resources in the namespace of the specified resource types")
	cmd.Flags().String("resource-version", "", "If non-empty, the labels update will only succeed if this is the current resource-version for the object. Only valid when specifying a single resource.")
	usage := "identifying the resource to update the labels"
	cmdutil.AddFilenameOptionFlags(cmd, &options.FilenameOptions, usage)
	cmdutil.AddDryRunFlag(cmd)
	cmdutil.AddRecordFlag(cmd)
	cmdutil.AddInclude3rdPartyFlags(cmd)

	return cmd
}

// Complete adapts from the command line args and factory to the data required.
func (o *LabelOptions) Complete(f cmdutil.Factory, out io.Writer, cmd *cobra.Command, args []string) (err error) {
	o.out = out
	o.local = cmdutil.GetFlagBool(cmd, "local")
	o.overwrite = cmdutil.GetFlagBool(cmd, "overwrite")
	o.all = cmdutil.GetFlagBool(cmd, "all")
	o.resourceVersion = cmdutil.GetFlagString(cmd, "resource-version")
	o.selector = cmdutil.GetFlagString(cmd, "selector")
	o.outputFormat = cmdutil.GetFlagString(cmd, "output")
	o.dryrun = cmdutil.GetDryRunFlag(cmd)

	resources, labelArgs, err := cmdutil.GetResourcesAndPairs(args, "label")
	if err != nil {
		return err
	}
	o.resources = resources
	o.newLabels, o.removeLabels, err = parseLabels(labelArgs)
	return err
}

// Validate checks to the LabelOptions to see if there is sufficient information run the command.
func (o *LabelOptions) Validate() error {
	if len(o.resources) < 1 && cmdutil.IsFilenameEmpty(o.FilenameOptions.Filenames) {
		return fmt.Errorf("one or more resources must be specified as <resource> <name> or <resource>/<name>")
	}
	if len(o.newLabels) < 1 && len(o.removeLabels) < 1 {
		return fmt.Errorf("at least one label update is required")
	}
	return nil
}

// RunLabel does the work
func (o *LabelOptions) RunLabel(f cmdutil.Factory, cmd *cobra.Command) error {
	cmdNamespace, enforceNamespace, err := f.DefaultNamespace()
	if err != nil {
		return err
	}

	changeCause := f.Command()
	mapper, typer := f.Object()
	b := resource.NewBuilder(mapper, typer, resource.ClientMapperFunc(f.ClientForMapping), f.Decoder(true)).
		ContinueOnError().
		NamespaceParam(cmdNamespace).DefaultNamespace().
		FilenameParam(enforceNamespace, &o.FilenameOptions).
		Flatten()

	if !o.local {
		b = b.SelectorParam(o.selector).
			ResourceTypeOrNameArgs(o.all, o.resources...).
			Latest()
	}
	one := false
	r := b.Do().IntoSingleItemImplied(&one)
	if err := r.Err(); err != nil {
		return err
	}

	// only apply resource version locking on a single resource
	if !one && len(o.resourceVersion) > 0 {
		return fmt.Errorf("--resource-version may only be used with a single resource")
	}

	// TODO: support bulk generic output a la Get
	return r.Visit(func(info *resource.Info, err error) error {
		if err != nil {
			return err
		}

		var outputObj runtime.Object
		dataChangeMsg := "not labeled"
		if o.dryrun || o.local {
			err = labelFunc(info.Object, o.overwrite, o.resourceVersion, o.newLabels, o.removeLabels)
			if err != nil {
				return err
			}
			outputObj = info.Object
		} else {
			obj, err := cmdutil.MaybeConvertObject(info.Object, info.Mapping.GroupVersionKind.GroupVersion(), info.Mapping)
			if err != nil {
				return err
			}
			name, namespace := info.Name, info.Namespace
			oldData, err := json.Marshal(obj)
			if err != nil {
				return err
			}
			accessor, err := meta.Accessor(obj)
			if err != nil {
				return err
			}
			for _, label := range o.removeLabels {
				if _, ok := accessor.GetLabels()[label]; !ok {
					fmt.Fprintf(o.out, "label %q not found.\n", label)
				}
			}

			if err := labelFunc(obj, o.overwrite, o.resourceVersion, o.newLabels, o.removeLabels); err != nil {
				return err
			}
			if cmdutil.ShouldRecord(cmd, info) {
				if err := cmdutil.RecordChangeCause(obj, changeCause); err != nil {
					return err
				}
			}
			newData, err := json.Marshal(obj)
			if err != nil {
				return err
			}
			if !reflect.DeepEqual(oldData, newData) {
				dataChangeMsg = "labeled"
			}
			patchBytes, err := strategicpatch.CreateTwoWayMergePatch(oldData, newData, obj)
			createdPatch := err == nil
			if err != nil {
				glog.V(2).Infof("couldn't compute patch: %v", err)
			}

			mapping := info.ResourceMapping()
			client, err := f.ClientForMapping(mapping)
			if err != nil {
				return err
			}
			helper := resource.NewHelper(client, mapping)

			if createdPatch {
				outputObj, err = helper.Patch(namespace, name, api.StrategicMergePatchType, patchBytes)
			} else {
				outputObj, err = helper.Replace(namespace, name, false, obj)
			}
			if err != nil {
				return err
			}
		}
		if o.outputFormat != "" {
			return f.PrintObject(cmd, mapper, outputObj, o.out)
		}
		cmdutil.PrintSuccess(mapper, false, o.out, info.Mapping.Resource, info.Name, o.dryrun, dataChangeMsg)
		return nil
	})
}

func validateNoOverwrites(accessor meta.Object, labels map[string]string) error {
	allErrs := []error{}
	for key := range labels {
		if value, found := accessor.GetLabels()[key]; found {
			allErrs = append(allErrs, fmt.Errorf("'%s' already has a value (%s), and --overwrite is false", key, value))
		}
	}
	return utilerrors.NewAggregate(allErrs)
}

func parseLabels(spec []string) (map[string]string, []string, error) {
	labels := map[string]string{}
	var remove []string
	for _, labelSpec := range spec {
		if strings.Index(labelSpec, "=") != -1 {
			parts := strings.Split(labelSpec, "=")
			if len(parts) != 2 {
				return nil, nil, fmt.Errorf("invalid label spec: %v", labelSpec)
			}
			if errs := validation.IsValidLabelValue(parts[1]); len(errs) != 0 {
				return nil, nil, fmt.Errorf("invalid label value: %q: %s", labelSpec, strings.Join(errs, ";"))
			}
			labels[parts[0]] = parts[1]
		} else if strings.HasSuffix(labelSpec, "-") {
			remove = append(remove, labelSpec[:len(labelSpec)-1])
		} else {
			return nil, nil, fmt.Errorf("unknown label spec: %v", labelSpec)
		}
	}
	for _, removeLabel := range remove {
		if _, found := labels[removeLabel]; found {
			return nil, nil, fmt.Errorf("can not both modify and remove a label in the same command")
		}
	}
	return labels, remove, nil
}

func labelFunc(obj runtime.Object, overwrite bool, resourceVersion string, labels map[string]string, remove []string) error {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return err
	}
	if !overwrite {
		if err := validateNoOverwrites(accessor, labels); err != nil {
			return err
		}
	}

	objLabels := accessor.GetLabels()
	if objLabels == nil {
		objLabels = make(map[string]string)
	}

	for key, value := range labels {
		objLabels[key] = value
	}
	for _, label := range remove {
		delete(objLabels, label)
	}
	accessor.SetLabels(objLabels)

	if len(resourceVersion) != 0 {
		accessor.SetResourceVersion(resourceVersion)
	}
	return nil
}
