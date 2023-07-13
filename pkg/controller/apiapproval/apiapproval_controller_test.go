// SPDX-License-Identifier: AGPL-3.0-only
// Provenance-includes-location: https://github.com/kubernetes/apiextensions-apiserver/blob/master/pkg/controller/apiapproval/apiapproval_controller_test.go
// Provenance-includes-license: Apache-2.0
// Provenance-includes-copyright: The Kubernetes Authors.

package apiapproval

import (
	"testing"

	kindsv1 "github.com/grafana/grafana-apiserver/pkg/apis/kinds/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCalculateCondition(t *testing.T) {
	noConditionFn := func(t *testing.T, condition *kindsv1.GrafanaResourceDefinitionCondition) {
		t.Helper()
		if condition != nil {
			t.Fatal(condition)
		}
	}

	verifyCondition := func(status kindsv1.ConditionStatus, message string) func(t *testing.T, condition *kindsv1.GrafanaResourceDefinitionCondition) {
		return func(t *testing.T, condition *kindsv1.GrafanaResourceDefinitionCondition) {
			t.Helper()
			if condition == nil {
				t.Fatal("missing condition")
			}
			if e, a := status, condition.Status; e != a {
				t.Errorf("expected %v, got %v", e, a)
			}
			if e, a := message, condition.Message; e != a {
				t.Errorf("expected %v, got %v", e, a)
			}
		}
	}

	tests := []struct {
		name string

		group             string
		annotationValue   string
		validateCondition func(t *testing.T, condition *kindsv1.GrafanaResourceDefinitionCondition)
	}{
		{
			name:              "for other group",
			group:             "other.io",
			annotationValue:   "",
			validateCondition: noConditionFn,
		},
		{
			name:              "missing annotation",
			group:             "sigs.k8s.io",
			annotationValue:   "",
			validateCondition: verifyCondition(kindsv1.ConditionFalse, `protected groups must have approval annotation "api-approved.kubernetes.io", see https://github.com/kubernetes/enhancements/pull/1111`),
		},
		{
			name:              "invalid annotation",
			group:             "sigs.k8s.io",
			annotationValue:   "bad value",
			validateCondition: verifyCondition(kindsv1.ConditionFalse, `protected groups must have approval annotation "api-approved.kubernetes.io" with either a URL or a reason starting with "unapproved", see https://github.com/kubernetes/enhancements/pull/1111`),
		},
		{
			name:              "approved",
			group:             "sigs.k8s.io",
			annotationValue:   "https://github.com/kubernetes/kubernetes/pull/79724",
			validateCondition: verifyCondition(kindsv1.ConditionTrue, `approved in https://github.com/kubernetes/kubernetes/pull/79724`),
		},
		{
			name:              "unapproved",
			group:             "sigs.k8s.io",
			annotationValue:   "unapproved for reasons",
			validateCondition: verifyCondition(kindsv1.ConditionFalse, `not approved: "unapproved for reasons"`),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			grd := &kindsv1.GrafanaResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Annotations: map[string]string{kindsv1.KubeAPIApprovedAnnotation: test.annotationValue}},
				Spec: kindsv1.GrafanaResourceDefinitionSpec{
					Group: test.group,
				},
			}

			actual := calculateCondition(grd)
			test.validateCondition(t, actual)

		})
	}

}
