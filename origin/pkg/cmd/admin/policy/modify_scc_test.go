package policy

import (
	"reflect"
	"testing"

	kapi "github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset/fake"
	"github.com/openshift/kubernetes/pkg/client/testing/core"
	"github.com/openshift/kubernetes/pkg/runtime"

	authorizationapi "github.com/openshift/origin/pkg/authorization/api"
)

func TestModifySCC(t *testing.T) {
	tests := map[string]struct {
		startingSCC *kapi.SecurityContextConstraints
		subjects    []kapi.ObjectReference
		expectedSCC *kapi.SecurityContextConstraints
		remove      bool
	}{
		"add-user-to-empty": {
			startingSCC: &kapi.SecurityContextConstraints{},
			subjects:    []kapi.ObjectReference{{Name: "one", Kind: authorizationapi.UserKind}, {Name: "two", Kind: authorizationapi.UserKind}},
			expectedSCC: &kapi.SecurityContextConstraints{Users: []string{"one", "two"}},
			remove:      false,
		},
		"add-user-to-existing": {
			startingSCC: &kapi.SecurityContextConstraints{Users: []string{"one"}},
			subjects:    []kapi.ObjectReference{{Name: "two", Kind: authorizationapi.UserKind}},
			expectedSCC: &kapi.SecurityContextConstraints{Users: []string{"one", "two"}},
			remove:      false,
		},
		"add-user-to-existing-with-overlap": {
			startingSCC: &kapi.SecurityContextConstraints{Users: []string{"one"}},
			subjects:    []kapi.ObjectReference{{Name: "one", Kind: authorizationapi.UserKind}, {Name: "two", Kind: authorizationapi.UserKind}},
			expectedSCC: &kapi.SecurityContextConstraints{Users: []string{"one", "two"}},
			remove:      false,
		},

		"add-sa-to-empty": {
			startingSCC: &kapi.SecurityContextConstraints{},
			subjects:    []kapi.ObjectReference{{Namespace: "a", Name: "one", Kind: authorizationapi.ServiceAccountKind}, {Namespace: "b", Name: "two", Kind: authorizationapi.ServiceAccountKind}},
			expectedSCC: &kapi.SecurityContextConstraints{Users: []string{"system:serviceaccount:a:one", "system:serviceaccount:b:two"}},
			remove:      false,
		},
		"add-sa-to-existing": {
			startingSCC: &kapi.SecurityContextConstraints{Users: []string{"one"}},
			subjects:    []kapi.ObjectReference{{Namespace: "b", Name: "two", Kind: authorizationapi.ServiceAccountKind}},
			expectedSCC: &kapi.SecurityContextConstraints{Users: []string{"one", "system:serviceaccount:b:two"}},
			remove:      false,
		},
		"add-sa-to-existing-with-overlap": {
			startingSCC: &kapi.SecurityContextConstraints{Users: []string{"system:serviceaccount:a:one"}},
			subjects:    []kapi.ObjectReference{{Namespace: "a", Name: "one", Kind: authorizationapi.ServiceAccountKind}, {Namespace: "b", Name: "two", Kind: authorizationapi.ServiceAccountKind}},
			expectedSCC: &kapi.SecurityContextConstraints{Users: []string{"system:serviceaccount:a:one", "system:serviceaccount:b:two"}},
			remove:      false,
		},

		"add-group-to-empty": {
			startingSCC: &kapi.SecurityContextConstraints{},
			subjects:    []kapi.ObjectReference{{Name: "one", Kind: authorizationapi.GroupKind}, {Name: "two", Kind: authorizationapi.GroupKind}},
			expectedSCC: &kapi.SecurityContextConstraints{Groups: []string{"one", "two"}},
			remove:      false,
		},
		"add-group-to-existing": {
			startingSCC: &kapi.SecurityContextConstraints{Groups: []string{"one"}},
			subjects:    []kapi.ObjectReference{{Name: "two", Kind: authorizationapi.GroupKind}},
			expectedSCC: &kapi.SecurityContextConstraints{Groups: []string{"one", "two"}},
			remove:      false,
		},
		"add-group-to-existing-with-overlap": {
			startingSCC: &kapi.SecurityContextConstraints{Groups: []string{"one"}},
			subjects:    []kapi.ObjectReference{{Name: "one", Kind: authorizationapi.GroupKind}, {Name: "two", Kind: authorizationapi.GroupKind}},
			expectedSCC: &kapi.SecurityContextConstraints{Groups: []string{"one", "two"}},
			remove:      false,
		},

		"remove-user": {
			startingSCC: &kapi.SecurityContextConstraints{Users: []string{"one", "two"}},
			subjects:    []kapi.ObjectReference{{Name: "one", Kind: authorizationapi.UserKind}, {Name: "two", Kind: authorizationapi.UserKind}},
			expectedSCC: &kapi.SecurityContextConstraints{},
			remove:      true,
		},
		"remove-user-from-existing-with-overlap": {
			startingSCC: &kapi.SecurityContextConstraints{Users: []string{"one", "two"}},
			subjects:    []kapi.ObjectReference{{Name: "two", Kind: authorizationapi.UserKind}},
			expectedSCC: &kapi.SecurityContextConstraints{Users: []string{"one"}},
			remove:      true,
		},

		"remove-sa": {
			startingSCC: &kapi.SecurityContextConstraints{Users: []string{"system:serviceaccount:a:one", "system:serviceaccount:b:two"}},
			subjects:    []kapi.ObjectReference{{Namespace: "a", Name: "one", Kind: authorizationapi.ServiceAccountKind}, {Namespace: "b", Name: "two", Kind: authorizationapi.ServiceAccountKind}},
			expectedSCC: &kapi.SecurityContextConstraints{},
			remove:      true,
		},
		"remove-sa-from-existing-with-overlap": {
			startingSCC: &kapi.SecurityContextConstraints{Users: []string{"system:serviceaccount:a:one", "system:serviceaccount:b:two"}},
			subjects:    []kapi.ObjectReference{{Namespace: "b", Name: "two", Kind: authorizationapi.ServiceAccountKind}},
			expectedSCC: &kapi.SecurityContextConstraints{Users: []string{"system:serviceaccount:a:one"}},
			remove:      true,
		},

		"remove-group": {
			startingSCC: &kapi.SecurityContextConstraints{Groups: []string{"one", "two"}},
			subjects:    []kapi.ObjectReference{{Name: "one", Kind: authorizationapi.GroupKind}, {Name: "two", Kind: authorizationapi.GroupKind}},
			expectedSCC: &kapi.SecurityContextConstraints{},
			remove:      true,
		},
		"remove-group-from-existing-with-overlap": {
			startingSCC: &kapi.SecurityContextConstraints{Groups: []string{"one", "two"}},
			subjects:    []kapi.ObjectReference{{Name: "two", Kind: authorizationapi.GroupKind}},
			expectedSCC: &kapi.SecurityContextConstraints{Groups: []string{"one"}},
			remove:      true,
		},
	}

	for tcName, tc := range tests {
		fakeClient := fake.NewSimpleClientset()
		fakeClient.PrependReactor("get", "securitycontextconstraints", func(action core.Action) (handled bool, ret runtime.Object, err error) {
			return true, tc.startingSCC, nil
		})
		var actualSCC *kapi.SecurityContextConstraints
		fakeClient.PrependReactor("update", "securitycontextconstraints", func(action core.Action) (handled bool, ret runtime.Object, err error) {
			actualSCC = action.(core.UpdateAction).GetObject().(*kapi.SecurityContextConstraints)
			return true, actualSCC, nil
		})

		o := &SCCModificationOptions{
			SCCName:                 "foo",
			SCCInterface:            fakeClient.Core(),
			DefaultSubjectNamespace: "",
			Subjects:                tc.subjects,
		}

		var err error
		if tc.remove {
			err = o.RemoveSCC()
		} else {
			err = o.AddSCC()
		}
		if err != nil {
			t.Errorf("%s: unexpected err %v", tcName, err)
		}
		if e, a := tc.expectedSCC.Users, actualSCC.Users; !reflect.DeepEqual(e, a) {
			t.Errorf("%s: expected %v, actual %v", tcName, e, a)
		}
		if e, a := tc.expectedSCC.Groups, actualSCC.Groups; !reflect.DeepEqual(e, a) {
			t.Errorf("%s: expected %v, actual %v", tcName, e, a)
		}
	}
}
