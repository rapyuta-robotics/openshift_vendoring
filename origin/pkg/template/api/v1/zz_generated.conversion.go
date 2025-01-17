// +build !ignore_autogenerated_openshift

// This file was autogenerated by conversion-gen. Do not edit it manually!

package v1

import (
	api "github.com/openshift/origin/pkg/template/api"
	api_v1 "github.com/openshift/kubernetes/pkg/api/v1"
	conversion "github.com/openshift/kubernetes/pkg/conversion"
	runtime "github.com/openshift/kubernetes/pkg/runtime"
	unsafe "unsafe"
)

func init() {
	SchemeBuilder.Register(RegisterConversions)
}

// RegisterConversions adds conversion functions to the given scheme.
// Public to allow building arbitrary schemes.
func RegisterConversions(scheme *runtime.Scheme) error {
	return scheme.AddGeneratedConversionFuncs(
		Convert_v1_Parameter_To_api_Parameter,
		Convert_api_Parameter_To_v1_Parameter,
		Convert_v1_Template_To_api_Template,
		Convert_api_Template_To_v1_Template,
		Convert_v1_TemplateList_To_api_TemplateList,
		Convert_api_TemplateList_To_v1_TemplateList,
	)
}

func autoConvert_v1_Parameter_To_api_Parameter(in *Parameter, out *api.Parameter, s conversion.Scope) error {
	out.Name = in.Name
	out.DisplayName = in.DisplayName
	out.Description = in.Description
	out.Value = in.Value
	out.Generate = in.Generate
	out.From = in.From
	out.Required = in.Required
	return nil
}

func Convert_v1_Parameter_To_api_Parameter(in *Parameter, out *api.Parameter, s conversion.Scope) error {
	return autoConvert_v1_Parameter_To_api_Parameter(in, out, s)
}

func autoConvert_api_Parameter_To_v1_Parameter(in *api.Parameter, out *Parameter, s conversion.Scope) error {
	out.Name = in.Name
	out.DisplayName = in.DisplayName
	out.Description = in.Description
	out.Value = in.Value
	out.Generate = in.Generate
	out.From = in.From
	out.Required = in.Required
	return nil
}

func Convert_api_Parameter_To_v1_Parameter(in *api.Parameter, out *Parameter, s conversion.Scope) error {
	return autoConvert_api_Parameter_To_v1_Parameter(in, out, s)
}

func autoConvert_v1_Template_To_api_Template(in *Template, out *api.Template, s conversion.Scope) error {
	if err := api_v1.Convert_v1_ObjectMeta_To_api_ObjectMeta(&in.ObjectMeta, &out.ObjectMeta, s); err != nil {
		return err
	}
	out.Message = in.Message
	if in.Objects != nil {
		in, out := &in.Objects, &out.Objects
		*out = make([]runtime.Object, len(*in))
		for i := range *in {
			if err := runtime.Convert_runtime_RawExtension_To_runtime_Object(&(*in)[i], &(*out)[i], s); err != nil {
				return err
			}
		}
	} else {
		out.Objects = nil
	}
	out.Parameters = *(*[]api.Parameter)(unsafe.Pointer(&in.Parameters))
	out.ObjectLabels = *(*map[string]string)(unsafe.Pointer(&in.ObjectLabels))
	return nil
}

func Convert_v1_Template_To_api_Template(in *Template, out *api.Template, s conversion.Scope) error {
	return autoConvert_v1_Template_To_api_Template(in, out, s)
}

func autoConvert_api_Template_To_v1_Template(in *api.Template, out *Template, s conversion.Scope) error {
	if err := api_v1.Convert_api_ObjectMeta_To_v1_ObjectMeta(&in.ObjectMeta, &out.ObjectMeta, s); err != nil {
		return err
	}
	out.Message = in.Message
	out.Parameters = *(*[]Parameter)(unsafe.Pointer(&in.Parameters))
	if in.Objects != nil {
		in, out := &in.Objects, &out.Objects
		*out = make([]runtime.RawExtension, len(*in))
		for i := range *in {
			if err := runtime.Convert_runtime_Object_To_runtime_RawExtension(&(*in)[i], &(*out)[i], s); err != nil {
				return err
			}
		}
	} else {
		out.Objects = nil
	}
	out.ObjectLabels = *(*map[string]string)(unsafe.Pointer(&in.ObjectLabels))
	return nil
}

func Convert_api_Template_To_v1_Template(in *api.Template, out *Template, s conversion.Scope) error {
	return autoConvert_api_Template_To_v1_Template(in, out, s)
}

func autoConvert_v1_TemplateList_To_api_TemplateList(in *TemplateList, out *api.TemplateList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]api.Template, len(*in))
		for i := range *in {
			if err := Convert_v1_Template_To_api_Template(&(*in)[i], &(*out)[i], s); err != nil {
				return err
			}
		}
	} else {
		out.Items = nil
	}
	return nil
}

func Convert_v1_TemplateList_To_api_TemplateList(in *TemplateList, out *api.TemplateList, s conversion.Scope) error {
	return autoConvert_v1_TemplateList_To_api_TemplateList(in, out, s)
}

func autoConvert_api_TemplateList_To_v1_TemplateList(in *api.TemplateList, out *TemplateList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Template, len(*in))
		for i := range *in {
			if err := Convert_api_Template_To_v1_Template(&(*in)[i], &(*out)[i], s); err != nil {
				return err
			}
		}
	} else {
		out.Items = nil
	}
	return nil
}

func Convert_api_TemplateList_To_v1_TemplateList(in *api.TemplateList, out *TemplateList, s conversion.Scope) error {
	return autoConvert_api_TemplateList_To_v1_TemplateList(in, out, s)
}
