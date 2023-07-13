// SPDX-License-Identifier: AGPL-3.0-only

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1 "github.com/grafana/grafana-apiserver/pkg/client/clientset/clientset/typed/kinds/v1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeKindsV1 struct {
	*testing.Fake
}

func (c *FakeKindsV1) GrafanaResourceDefinitions() v1.GrafanaResourceDefinitionInterface {
	return &FakeGrafanaResourceDefinitions{c}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeKindsV1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}