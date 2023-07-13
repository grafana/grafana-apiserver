// SPDX-License-Identifier: AGPL-3.0-only

package runtime

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var _ runtime.ObjectTyper = (*objectTyper)(nil)

type objectTyper struct{}

func NewObjectTyper() *objectTyper {
	return &objectTyper{}
}

func (dt *objectTyper) ObjectKinds(obj runtime.Object) ([]schema.GroupVersionKind, bool, error) {
	return []schema.GroupVersionKind{obj.GetObjectKind().GroupVersionKind()}, false, nil
}

func (dt *objectTyper) Recognizes(gvk schema.GroupVersionKind) bool {
	return true
}
