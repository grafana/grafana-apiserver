// SPDX-License-Identifier: AGPL-3.0-only
// Provenance-includes-location: https://github.com/kubernetes/apiextensions-apiserver/blob/master/pkg/apis/apiextensions/v1/conversion.go
// Provenance-includes-license: Apache-2.0
// Provenance-includes-copyright: The Kubernetes Authors.

package v1

import (
	"github.com/grafana/grafana-apiserver/pkg/apis/kinds"
	"k8s.io/apimachinery/pkg/conversion"
)

func Convert_kinds_GrafanaResourceDefinitionSpec_To_v1_GrafanaResourceDefinitionSpec(in *kinds.GrafanaResourceDefinitionSpec, out *GrafanaResourceDefinitionSpec, s conversion.Scope) error {
	if err := autoConvert_kinds_GrafanaResourceDefinitionSpec_To_v1_GrafanaResourceDefinitionSpec(in, out, s); err != nil {
		return err
	}

	if len(out.Versions) == 0 && len(in.Version) > 0 {
		// no versions were specified, and a version name was specified
		out.Versions = []GrafanaResourceDefinitionVersion{{Name: in.Version, Served: true, Storage: true}}
	}

	// If spec.{subresources,validation,additionalPrinterColumns} exists, move to versions
	if in.Subresources != nil {
		subresources := &GrafanaResourceSubresources{}
		if err := Convert_kinds_GrafanaResourceSubresources_To_v1_GrafanaResourceSubresources(in.Subresources, subresources, s); err != nil {
			return err
		}
		for i := range out.Versions {
			out.Versions[i].Subresources = subresources
		}
	}
	return nil
}
