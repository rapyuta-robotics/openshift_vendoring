package config

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/openshift/github.com/evanphx/json-patch"
	"github.com/openshift/github.com/spf13/cobra"

	"github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/kubectl"
	cmdutil "github.com/openshift/kubernetes/pkg/kubectl/cmd/util"
	"github.com/openshift/kubernetes/pkg/kubectl/resource"
	"github.com/openshift/kubernetes/pkg/runtime"
	"github.com/openshift/kubernetes/pkg/util/sets"
	"github.com/openshift/kubernetes/pkg/util/strategicpatch"
	"github.com/openshift/kubernetes/pkg/util/yaml"

	configapi "github.com/openshift/origin/pkg/cmd/server/api"
	configapiinstall "github.com/openshift/origin/pkg/cmd/server/api/install"
	"github.com/openshift/origin/pkg/cmd/templates"
	"github.com/openshift/origin/pkg/cmd/util/clientcmd"
)

const PatchRecommendedName = "patch"

var patchTypes = map[string]api.PatchType{"json": api.JSONPatchType, "merge": api.MergePatchType, "strategic": api.StrategicMergePatchType}

// PatchOptions is the start of the data required to perform the operation.  As new fields are added, add them here instead of
// referencing the cmd.Flags()
type PatchOptions struct {
	Filename  string
	Patch     string
	PatchType api.PatchType

	Builder *resource.Builder

	Out io.Writer
}

var (
	patch_long    = templates.LongDesc(`Patch the master-config.yaml or node-config.yaml`)
	patch_example = templates.Examples(`
		# Set the auditConfig.enabled value to true
		%[1]s openshift.local.config/master/master-config.yaml --patch='{"auditConfig": {"enabled": true}}'`)
)

func NewCmdPatch(name, fullName string, f *clientcmd.Factory, out io.Writer) *cobra.Command {
	o := &PatchOptions{Out: out}

	cmd := &cobra.Command{
		Use:     name + " FILENAME -p PATCH",
		Short:   "Update field(s) of a resource using a patch.",
		Long:    patch_long,
		Example: patch_example,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(o.Complete(f, cmd, args))
			cmdutil.CheckErr(o.Validate())
			cmdutil.CheckErr(o.RunPatch())
		},
	}
	cmd.Flags().StringVarP(&o.Patch, "patch", "p", "", "The patch to be applied to the resource JSON file.")
	cmd.MarkFlagRequired("patch")
	cmd.Flags().String("type", "strategic", fmt.Sprintf("The type of patch being provided; one of %v", sets.StringKeySet(patchTypes).List()))

	return cmd
}

func (o *PatchOptions) Complete(f *clientcmd.Factory, cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("exactly one FILENAME is allowed: %v", args)
	}
	o.Filename = args[0]

	patchTypeString := strings.ToLower(cmdutil.GetFlagString(cmd, "type"))
	ok := false
	o.PatchType, ok = patchTypes[patchTypeString]
	if !ok {
		return cmdutil.UsageError(cmd, fmt.Sprintf("--type must be one of %v, not %q", sets.StringKeySet(patchTypes).List(), patchTypeString))
	}

	o.Builder = resource.NewBuilder(configapiinstall.NewRESTMapper(), configapi.Scheme, resource.DisabledClientForMapping{}, configapi.Codecs.LegacyCodec())

	return nil
}

func (o *PatchOptions) Validate() error {
	if len(o.Patch) == 0 {
		return errors.New("must specify -p to patch")
	}
	if len(o.Filename) == 0 {
		return errors.New("filename is required")
	}

	return nil
}

func (o *PatchOptions) RunPatch() error {
	patchBytes, err := yaml.ToJSON([]byte(o.Patch))
	if err != nil {
		return fmt.Errorf("unable to parse %q: %v", o.Patch, err)
	}

	r := o.Builder.
		FilenameParam(false, &resource.FilenameOptions{Recursive: false, Filenames: []string{o.Filename}}).
		Flatten().
		Do()
	err = r.Err()
	if err != nil {
		return err
	}

	infos, err := r.Infos()
	if err != nil {
		return err
	}
	if len(infos) > 1 {
		return fmt.Errorf("multiple resources provided")
	}
	info := infos[0]

	originalObjJS, err := runtime.Encode(configapi.Codecs.LegacyCodec(info.Mapping.GroupVersionKind.GroupVersion()), info.VersionedObject.(runtime.Object))
	if err != nil {
		return err
	}
	patchedObj, err := configapi.Scheme.DeepCopy(info.VersionedObject)
	if err != nil {
		return err
	}
	originalPatchedObjJS, err := getPatchedJS(o.PatchType, originalObjJS, patchBytes, patchedObj.(runtime.Object))
	if err != nil {
		return err
	}

	rawExtension := &runtime.Unknown{
		Raw: originalPatchedObjJS,
	}
	printer, _, err := kubectl.GetPrinter("yaml", "", false, true)
	if err != nil {
		return err
	}
	if err := printer.PrintObj(rawExtension, o.Out); err != nil {
		return err
	}

	return nil
}

func getPatchedJS(patchType api.PatchType, originalJS, patchJS []byte, obj runtime.Object) ([]byte, error) {
	switch patchType {
	case api.JSONPatchType:
		patchObj, err := jsonpatch.DecodePatch(patchJS)
		if err != nil {
			return nil, err
		}
		return patchObj.Apply(originalJS)

	case api.MergePatchType:
		return jsonpatch.MergePatch(originalJS, patchJS)

	case api.StrategicMergePatchType:
		return strategicpatch.StrategicMergePatchData(originalJS, patchJS, obj)

	default:
		// only here as a safety net - go-restful filters content-type
		return nil, fmt.Errorf("unknown Content-Type header for patch: %v", patchType)
	}
}
