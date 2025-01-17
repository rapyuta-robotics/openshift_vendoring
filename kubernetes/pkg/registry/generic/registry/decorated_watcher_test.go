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

package registry

import (
	"fmt"
	"testing"
	"time"

	"github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/runtime"
	"github.com/openshift/kubernetes/pkg/util/wait"
	"github.com/openshift/kubernetes/pkg/watch"
)

func TestDecoratedWatcher(t *testing.T) {
	w := watch.NewFake()
	decorator := func(obj runtime.Object) error {
		pod := obj.(*api.Pod)
		pod.Annotations = map[string]string{"decorated": "true"}
		return nil
	}
	dw := newDecoratedWatcher(w, decorator)
	defer dw.Stop()

	go w.Add(&api.Pod{ObjectMeta: api.ObjectMeta{Name: "foo"}})
	select {
	case e := <-dw.ResultChan():
		pod, ok := e.Object.(*api.Pod)
		if !ok {
			t.Errorf("Should received object of type *api.Pod, get type (%T)", e.Object)
			return
		}
		if pod.Annotations["decorated"] != "true" {
			t.Errorf("pod.Annotations[\"decorated\"], want=%s, get=%s", "true", pod.Labels["decorated"])
		}
	case <-time.After(wait.ForeverTestTimeout):
		t.Errorf("timeout after %v", wait.ForeverTestTimeout)
	}
}

func TestDecoratedWatcherError(t *testing.T) {
	w := watch.NewFake()
	expErr := fmt.Errorf("expected error")
	decorator := func(obj runtime.Object) error {
		return expErr
	}
	dw := newDecoratedWatcher(w, decorator)
	defer dw.Stop()

	go w.Add(&api.Pod{ObjectMeta: api.ObjectMeta{Name: "foo"}})
	select {
	case e := <-dw.ResultChan():
		if e.Type != watch.Error {
			t.Errorf("event type want=%v, get=%v", watch.Error, e.Type)
		}
	case <-time.After(wait.ForeverTestTimeout):
		t.Errorf("timeout after %v", wait.ForeverTestTimeout)
	}
}
