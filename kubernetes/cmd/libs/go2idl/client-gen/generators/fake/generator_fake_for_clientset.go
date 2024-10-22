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

package fake

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/openshift/k8s.io/gengo/generator"
	"github.com/openshift/k8s.io/gengo/namer"
	"github.com/openshift/k8s.io/gengo/types"
	clientgentypes "github.com/openshift/kubernetes/cmd/libs/go2idl/client-gen/types"
)

// genClientset generates a package for a clientset.
type genClientset struct {
	generator.DefaultGen
	groups             []clientgentypes.GroupVersions
	typedClientPath    string
	outputPackage      string
	imports            namer.ImportTracker
	clientsetGenerated bool
	// the import path of the generated real clientset.
	clientsetPath string
}

var _ generator.Generator = &genClientset{}

func (g *genClientset) Namers(c *generator.Context) namer.NameSystems {
	return namer.NameSystems{
		"raw": namer.NewRawNamer(g.outputPackage, g.imports),
	}
}

// We only want to call GenerateType() once.
func (g *genClientset) Filter(c *generator.Context, t *types.Type) bool {
	ret := !g.clientsetGenerated
	g.clientsetGenerated = true
	return ret
}

func (g *genClientset) Imports(c *generator.Context) (imports []string) {
	imports = append(imports, g.imports.ImportLines()...)
	for _, group := range g.groups {
		for _, version := range group.Versions {
			typedClientPath := filepath.Join(g.typedClientPath, group.Group.NonEmpty(), version.NonEmpty())
			imports = append(imports, strings.ToLower(fmt.Sprintf("%s%s \"%s\"", version.NonEmpty(), group.Group.NonEmpty(), typedClientPath)))
			fakeTypedClientPath := filepath.Join(typedClientPath, "fake")
			imports = append(imports, strings.ToLower(fmt.Sprintf("fake%s%s \"%s\"", version.NonEmpty(), group.Group.NonEmpty(), fakeTypedClientPath)))
		}
	}
	// the package that has the clientset Interface
	imports = append(imports, fmt.Sprintf("clientset \"%s\"", g.clientsetPath))
	// imports for the code in commonTemplate
	imports = append(imports,
		"github.com/openshift/kubernetes/pkg/api",
		"github.com/openshift/kubernetes/pkg/apimachinery/registered",
		"github.com/openshift/kubernetes/pkg/client/testing/core",
		"github.com/openshift/kubernetes/pkg/client/typed/discovery",
		"fakediscovery \"github.com/openshift/kubernetes/pkg/client/typed/discovery/fake\"",
		"github.com/openshift/kubernetes/pkg/runtime",
		"github.com/openshift/kubernetes/pkg/watch",
	)

	return
}

func (g *genClientset) GenerateType(c *generator.Context, t *types.Type, w io.Writer) error {
	// TODO: We actually don't need any type information to generate the clientset,
	// perhaps we can adapt the go2ild framework to this kind of usage.
	sw := generator.NewSnippetWriter(w, c, "$", "$")

	sw.Do(common, nil)

	sw.Do(checkImpl, nil)

	allGroups := clientgentypes.ToGroupVersionPackages(g.groups)

	for _, g := range allGroups {
		sw.Do(clientsetInterfaceImplTemplate, g)
		// don't generated the default method if generating internalversion clientset
		if g.IsDefaultVersion && g.Version != "" {
			sw.Do(clientsetInterfaceDefaultVersionImpl, g)
		}
	}

	return sw.Error()
}

// This part of code is version-independent, unchanging.
var common = `
// NewSimpleClientset returns a clientset that will respond with the provided objects.
// It's backed by a very simple object tracker that processes creates, updates and deletions as-is,
// without applying any validations and/or defaults. It shouldn't be considered a replacement
// for a real clientset and is mostly useful in simple unit tests.
func NewSimpleClientset(objects ...runtime.Object) *Clientset {
	o := core.NewObjectTracker(api.Scheme, api.Codecs.UniversalDecoder())
	for _, obj := range objects {
		if err := o.Add(obj); err != nil {
			panic(err)
		}
	}

	fakePtr := core.Fake{}
	fakePtr.AddReactor("*", "*", core.ObjectReaction(o, registered.RESTMapper()))

	fakePtr.AddWatchReactor("*", core.DefaultWatchReactor(watch.NewFake(), nil))

	return &Clientset{fakePtr}
}

// Clientset implements clientset.Interface. Meant to be embedded into a
// struct to get a default implementation. This makes faking out just the method
// you want to test easier.
type Clientset struct {
	core.Fake
}

func (c *Clientset) Discovery() discovery.DiscoveryInterface {
	return &fakediscovery.FakeDiscovery{Fake: &c.Fake}
}
`

var checkImpl = `
var _ clientset.Interface = &Clientset{}
`

var clientsetInterfaceImplTemplate = `
// $.GroupVersion$ retrieves the $.GroupVersion$Client
func (c *Clientset) $.GroupVersion$() $.PackageName$.$.GroupVersion$Interface {
	return &fake$.PackageName$.Fake$.GroupVersion${Fake: &c.Fake}
}
`

var clientsetInterfaceDefaultVersionImpl = `
// $.Group$ retrieves the $.GroupVersion$Client
func (c *Clientset) $.Group$() $.PackageName$.$.GroupVersion$Interface {
	return &fake$.PackageName$.Fake$.GroupVersion${Fake: &c.Fake}
}
`
