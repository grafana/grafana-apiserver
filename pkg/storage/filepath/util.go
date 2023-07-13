// SPDX-License-Identifier: AGPL-3.0-only
// Provenance-includes-location: https://github.com/kubernetes-sigs/apiserver-runtime/blob/main/pkg/experimental/storage/filepath/jsonfile_rest.go
// Provenance-includes-license: Apache-2.0
// Provenance-includes-copyright: The Kubernetes Authors.

package filepath

import (
	"bytes"
	"context"
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"
)

// objectFilePath returns the file path for the given object.
// The directory path is based on the object's GroupVersionKind and namespace:
// {root}/{Group}/{Version}/{Namespace}/{Kind}/{Name}.json
//
// If the object doesn't have a namespace, the file path is based on the
// object's GroupVersionKind:
// {root}/{Group}/{Version}/{Kind}/{Name}.json
func (s *Storage) objectFilePath(ctx context.Context, obj runtime.Object) (string, error) {
	dir := s.objectDirPath(ctx, obj)
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return "", err
	}
	name := accessor.GetName()

	return filepath.Join(dir, name+".json"), nil
}

// objectDirPath returns the directory path for the given object.
// The directory path is based on the object's GroupVersionKind and namespace:
// {root}/{Group}/{Version}/{Namespace}/{Kind}
//
// If the object doesn't have a namespace, the directory path is based on the
// object's GroupVersionKind:
// {root}/{Group}/{Version}/{Kind}
func (s *Storage) objectDirPath(ctx context.Context, obj runtime.Object) string {
	gvk := obj.GetObjectKind().GroupVersionKind()
	p := filepath.Join(s.root, gvk.Group, gvk.Version)

	if ns, ok := request.NamespaceFrom(ctx); ok {
		p = filepath.Join(p, ns)
	}

	return filepath.Join(p, gvk.Kind)
}

func writeFile(codec runtime.Codec, path string, obj runtime.Object) error {
	buf := new(bytes.Buffer)
	if err := codec.Encode(obj, buf); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0600)
}

func readFile(codec runtime.Codec, path string, newFunc func() runtime.Object) (runtime.Object, error) {
	content, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	newObj := newFunc()
	decodedObj, _, err := codec.Decode(content, nil, newObj)
	if err != nil {
		return nil, err
	}
	return decodedObj, nil
}

func deleteFile(path string) error {
	return os.Remove(path)
}

func exists(filepath string) bool {
	_, err := os.Stat(filepath)
	return err == nil
}

func ensureDir(dirname string) error {
	if !exists(dirname) {
		return os.MkdirAll(dirname, 0700)
	}
	return nil
}

func isUnchanged(codec runtime.Codec, obj runtime.Object, newObj runtime.Object) (bool, error) {
	buf := new(bytes.Buffer)
	if err := codec.Encode(obj, buf); err != nil {
		return false, err
	}

	newBuf := new(bytes.Buffer)
	if err := codec.Encode(newObj, newBuf); err != nil {
		return false, err
	}

	return bytes.Equal(buf.Bytes(), newBuf.Bytes()), nil
}
