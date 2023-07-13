// SPDX-License-Identifier: AGPL-3.0-only
// Provenance-includes-location: https://github.com/kubernetes/apiextensions-apiserver/blob/master/pkg/controller/status/naming_controller_test.go
// Provenance-includes-license: Apache-2.0
// Provenance-includes-copyright: The Kubernetes Authors.

package status

import (
	"reflect"
	"strings"
	"testing"
	"time"

	kindshelpers "github.com/grafana/grafana-apiserver/pkg/apihelpers"
	kindsv1 "github.com/grafana/grafana-apiserver/pkg/apis/kinds/v1"
	listers "github.com/grafana/grafana-apiserver/pkg/client/listers/kinds/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

type grdBuilder struct {
	curr kindsv1.GrafanaResourceDefinition
}

func newGRD(name string) *grdBuilder {
	tokens := strings.SplitN(name, ".", 2)
	return &grdBuilder{
		curr: kindsv1.GrafanaResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Spec: kindsv1.GrafanaResourceDefinitionSpec{
				Group: tokens[1],
				Names: kindsv1.GrafanaResourceDefinitionNames{
					Plural: tokens[0],
				},
			},
		},
	}
}

func (b *grdBuilder) SpecNames(plural, singular, kind, listKind string, shortNames ...string) *grdBuilder {
	b.curr.Spec.Names.Plural = plural
	b.curr.Spec.Names.Singular = singular
	b.curr.Spec.Names.Kind = kind
	b.curr.Spec.Names.ListKind = listKind
	b.curr.Spec.Names.ShortNames = shortNames

	return b
}

func (b *grdBuilder) StatusNames(plural, singular, kind, listKind string, shortNames ...string) *grdBuilder {
	b.curr.Status.AcceptedNames.Plural = plural
	b.curr.Status.AcceptedNames.Singular = singular
	b.curr.Status.AcceptedNames.Kind = kind
	b.curr.Status.AcceptedNames.ListKind = listKind
	b.curr.Status.AcceptedNames.ShortNames = shortNames

	return b
}

func (b *grdBuilder) Condition(c kindsv1.GrafanaResourceDefinitionCondition) *grdBuilder {
	b.curr.Status.Conditions = append(b.curr.Status.Conditions, c)

	return b
}

func names(plural, singular, kind, listKind string, shortNames ...string) kindsv1.GrafanaResourceDefinitionNames {
	ret := kindsv1.GrafanaResourceDefinitionNames{
		Plural:     plural,
		Singular:   singular,
		Kind:       kind,
		ListKind:   listKind,
		ShortNames: shortNames,
	}
	return ret
}

func (b *grdBuilder) NewOrDie() *kindsv1.GrafanaResourceDefinition {
	return &b.curr
}

var acceptedCondition = kindsv1.GrafanaResourceDefinitionCondition{
	Type:    kindsv1.NamesAccepted,
	Status:  kindsv1.ConditionTrue,
	Reason:  "NoConflicts",
	Message: "no conflicts found",
}

var notAcceptedCondition = kindsv1.GrafanaResourceDefinitionCondition{
	Type:    kindsv1.NamesAccepted,
	Status:  kindsv1.ConditionFalse,
	Reason:  "NotAccepted",
	Message: "not all names are accepted",
}

var installingCondition = kindsv1.GrafanaResourceDefinitionCondition{
	Type:    kindsv1.Established,
	Status:  kindsv1.ConditionFalse,
	Reason:  "Installing",
	Message: "the initial names have been accepted",
}

var notEstablishedCondition = kindsv1.GrafanaResourceDefinitionCondition{
	Type:    kindsv1.Established,
	Status:  kindsv1.ConditionFalse,
	Reason:  "NotAccepted",
	Message: "not all names are accepted",
}

func nameConflictCondition(reason, message string) kindsv1.GrafanaResourceDefinitionCondition {
	return kindsv1.GrafanaResourceDefinitionCondition{
		Type:    kindsv1.NamesAccepted,
		Status:  kindsv1.ConditionFalse,
		Reason:  reason,
		Message: message,
	}
}

func TestSync(t *testing.T) {
	tests := []struct {
		name string

		in                            *kindsv1.GrafanaResourceDefinition
		existing                      []*kindsv1.GrafanaResourceDefinition
		expectedNames                 kindsv1.GrafanaResourceDefinitionNames
		expectedNameConflictCondition kindsv1.GrafanaResourceDefinitionCondition
		expectedEstablishedCondition  kindsv1.GrafanaResourceDefinitionCondition
	}{
		{
			name:     "first resource",
			in:       newGRD("alfa.bravo.com").NewOrDie(),
			existing: []*kindsv1.GrafanaResourceDefinition{},
			expectedNames: kindsv1.GrafanaResourceDefinitionNames{
				Plural: "alfa",
			},
			expectedNameConflictCondition: acceptedCondition,
			expectedEstablishedCondition:  installingCondition,
		},
		{
			name: "different groups",
			in:   newGRD("alfa.bravo.com").SpecNames("alfa", "delta-singular", "echo-kind", "foxtrot-listkind", "golf-shortname-1", "hotel-shortname-2").NewOrDie(),
			existing: []*kindsv1.GrafanaResourceDefinition{
				newGRD("alfa.charlie.com").StatusNames("alfa", "delta-singular", "echo-kind", "foxtrot-listkind", "golf-shortname-1", "hotel-shortname-2").NewOrDie(),
			},
			expectedNames:                 names("alfa", "delta-singular", "echo-kind", "foxtrot-listkind", "golf-shortname-1", "hotel-shortname-2"),
			expectedNameConflictCondition: acceptedCondition,
			expectedEstablishedCondition:  installingCondition,
		},
		{
			name: "conflict plural to singular",
			in:   newGRD("alfa.bravo.com").SpecNames("alfa", "delta-singular", "echo-kind", "foxtrot-listkind", "golf-shortname-1", "hotel-shortname-2").NewOrDie(),
			existing: []*kindsv1.GrafanaResourceDefinition{
				newGRD("india.bravo.com").StatusNames("india", "alfa", "", "").NewOrDie(),
			},
			expectedNames:                 names("", "delta-singular", "echo-kind", "foxtrot-listkind", "golf-shortname-1", "hotel-shortname-2"),
			expectedNameConflictCondition: nameConflictCondition("PluralConflict", `"alfa" is already in use`),
			expectedEstablishedCondition:  notEstablishedCondition,
		},
		{
			name: "conflict singular to shortName",
			in:   newGRD("alfa.bravo.com").SpecNames("alfa", "delta-singular", "echo-kind", "foxtrot-listkind", "golf-shortname-1", "hotel-shortname-2").NewOrDie(),
			existing: []*kindsv1.GrafanaResourceDefinition{
				newGRD("india.bravo.com").StatusNames("india", "indias", "", "", "delta-singular").NewOrDie(),
			},
			expectedNames:                 names("alfa", "", "echo-kind", "foxtrot-listkind", "golf-shortname-1", "hotel-shortname-2"),
			expectedNameConflictCondition: nameConflictCondition("SingularConflict", `"delta-singular" is already in use`),
			expectedEstablishedCondition:  notEstablishedCondition,
		},
		{
			name: "conflict on shortName to shortName",
			in:   newGRD("alfa.bravo.com").SpecNames("alfa", "delta-singular", "echo-kind", "foxtrot-listkind", "golf-shortname-1", "hotel-shortname-2").NewOrDie(),
			existing: []*kindsv1.GrafanaResourceDefinition{
				newGRD("india.bravo.com").StatusNames("india", "indias", "", "", "hotel-shortname-2").NewOrDie(),
			},
			expectedNames:                 names("alfa", "delta-singular", "echo-kind", "foxtrot-listkind"),
			expectedNameConflictCondition: nameConflictCondition("ShortNamesConflict", `"hotel-shortname-2" is already in use`),
			expectedEstablishedCondition:  notEstablishedCondition,
		},
		{
			name: "conflict on kind to listkind",
			in:   newGRD("alfa.bravo.com").SpecNames("alfa", "delta-singular", "echo-kind", "foxtrot-listkind", "golf-shortname-1", "hotel-shortname-2").NewOrDie(),
			existing: []*kindsv1.GrafanaResourceDefinition{
				newGRD("india.bravo.com").StatusNames("india", "indias", "", "echo-kind").NewOrDie(),
			},
			expectedNames:                 names("alfa", "delta-singular", "", "foxtrot-listkind", "golf-shortname-1", "hotel-shortname-2"),
			expectedNameConflictCondition: nameConflictCondition("KindConflict", `"echo-kind" is already in use`),
			expectedEstablishedCondition:  notEstablishedCondition,
		},
		{
			name: "conflict on listkind to kind",
			in:   newGRD("alfa.bravo.com").SpecNames("alfa", "delta-singular", "echo-kind", "foxtrot-listkind", "golf-shortname-1", "hotel-shortname-2").NewOrDie(),
			existing: []*kindsv1.GrafanaResourceDefinition{
				newGRD("india.bravo.com").StatusNames("india", "indias", "foxtrot-listkind", "").NewOrDie(),
			},
			expectedNames:                 names("alfa", "delta-singular", "echo-kind", "", "golf-shortname-1", "hotel-shortname-2"),
			expectedNameConflictCondition: nameConflictCondition("ListKindConflict", `"foxtrot-listkind" is already in use`),
			expectedEstablishedCondition:  notEstablishedCondition,
		},
		{
			name: "no conflict on resource and kind",
			in:   newGRD("alfa.bravo.com").SpecNames("alfa", "delta-singular", "echo-kind", "foxtrot-listkind", "golf-shortname-1", "hotel-shortname-2").NewOrDie(),
			existing: []*kindsv1.GrafanaResourceDefinition{
				newGRD("india.bravo.com").StatusNames("india", "echo-kind", "", "").NewOrDie(),
			},
			expectedNames:                 names("alfa", "delta-singular", "echo-kind", "foxtrot-listkind", "golf-shortname-1", "hotel-shortname-2"),
			expectedNameConflictCondition: acceptedCondition,
			expectedEstablishedCondition:  installingCondition,
		},
		{
			name: "merge on conflicts",
			in: newGRD("alfa.bravo.com").
				SpecNames("alfa", "delta-singular", "echo-kind", "foxtrot-listkind", "golf-shortname-1", "hotel-shortname-2").
				StatusNames("zulu", "yankee-singular", "xray-kind", "whiskey-listkind", "victor-shortname-1", "uniform-shortname-2").
				NewOrDie(),
			existing: []*kindsv1.GrafanaResourceDefinition{
				newGRD("india.bravo.com").StatusNames("india", "indias", "foxtrot-listkind", "", "delta-singular").NewOrDie(),
			},
			expectedNames:                 names("alfa", "yankee-singular", "echo-kind", "whiskey-listkind", "golf-shortname-1", "hotel-shortname-2"),
			expectedNameConflictCondition: nameConflictCondition("ListKindConflict", `"foxtrot-listkind" is already in use`),
			expectedEstablishedCondition:  notEstablishedCondition,
		},
		{
			name: "merge on conflicts shortNames as one",
			in: newGRD("alfa.bravo.com").
				SpecNames("alfa", "delta-singular", "echo-kind", "foxtrot-listkind", "golf-shortname-1", "hotel-shortname-2").
				StatusNames("zulu", "yankee-singular", "xray-kind", "whiskey-listkind", "victor-shortname-1", "uniform-shortname-2").
				NewOrDie(),
			existing: []*kindsv1.GrafanaResourceDefinition{
				newGRD("india.bravo.com").StatusNames("india", "indias", "foxtrot-listkind", "", "delta-singular", "golf-shortname-1").NewOrDie(),
			},
			expectedNames:                 names("alfa", "yankee-singular", "echo-kind", "whiskey-listkind", "victor-shortname-1", "uniform-shortname-2"),
			expectedNameConflictCondition: nameConflictCondition("ListKindConflict", `"foxtrot-listkind" is already in use`),
			expectedEstablishedCondition:  notEstablishedCondition,
		},
		{
			name: "no conflicts on self",
			in: newGRD("alfa.bravo.com").
				SpecNames("alfa", "delta-singular", "echo-kind", "foxtrot-listkind", "golf-shortname-1", "hotel-shortname-2").
				StatusNames("alfa", "delta-singular", "echo-kind", "foxtrot-listkind", "golf-shortname-1", "hotel-shortname-2").
				NewOrDie(),
			existing: []*kindsv1.GrafanaResourceDefinition{
				newGRD("alfa.bravo.com").
					SpecNames("alfa", "delta-singular", "echo-kind", "foxtrot-listkind", "golf-shortname-1", "hotel-shortname-2").
					StatusNames("alfa", "delta-singular", "echo-kind", "foxtrot-listkind", "golf-shortname-1", "hotel-shortname-2").
					NewOrDie(),
			},
			expectedNames:                 names("alfa", "delta-singular", "echo-kind", "foxtrot-listkind", "golf-shortname-1", "hotel-shortname-2"),
			expectedNameConflictCondition: acceptedCondition,
			expectedEstablishedCondition:  installingCondition,
		},
		{
			name: "no conflicts on self, remove shortname",
			in: newGRD("alfa.bravo.com").
				SpecNames("alfa", "delta-singular", "echo-kind", "foxtrot-listkind", "golf-shortname-1").
				StatusNames("alfa", "delta-singular", "echo-kind", "foxtrot-listkind", "golf-shortname-1", "hotel-shortname-2").
				NewOrDie(),
			existing: []*kindsv1.GrafanaResourceDefinition{
				newGRD("alfa.bravo.com").
					SpecNames("alfa", "delta-singular", "echo-kind", "foxtrot-listkind", "golf-shortname-1", "hotel-shortname-2").
					StatusNames("alfa", "delta-singular", "echo-kind", "foxtrot-listkind", "golf-shortname-1", "hotel-shortname-2").
					NewOrDie(),
			},
			expectedNames:                 names("alfa", "delta-singular", "echo-kind", "foxtrot-listkind", "golf-shortname-1"),
			expectedNameConflictCondition: acceptedCondition,
			expectedEstablishedCondition:  installingCondition,
		},
		{
			name:     "installing before with true condition",
			in:       newGRD("alfa.bravo.com").Condition(acceptedCondition).NewOrDie(),
			existing: []*kindsv1.GrafanaResourceDefinition{},
			expectedNames: kindsv1.GrafanaResourceDefinitionNames{
				Plural: "alfa",
			},
			expectedNameConflictCondition: acceptedCondition,
			expectedEstablishedCondition:  installingCondition,
		},
		{
			name:     "not installing before with false condition",
			in:       newGRD("alfa.bravo.com").Condition(notAcceptedCondition).NewOrDie(),
			existing: []*kindsv1.GrafanaResourceDefinition{},
			expectedNames: kindsv1.GrafanaResourceDefinitionNames{
				Plural: "alfa",
			},
			expectedNameConflictCondition: acceptedCondition,
			expectedEstablishedCondition:  installingCondition,
		},
		{
			name: "conflicting, installing before with true condition",
			in: newGRD("alfa.bravo.com").SpecNames("alfa", "delta-singular", "echo-kind", "foxtrot-listkind", "golf-shortname-1", "hotel-shortname-2").
				Condition(acceptedCondition).
				NewOrDie(),
			existing: []*kindsv1.GrafanaResourceDefinition{
				newGRD("india.bravo.com").StatusNames("india", "alfa", "", "").NewOrDie(),
			},
			expectedNames:                 names("", "delta-singular", "echo-kind", "foxtrot-listkind", "golf-shortname-1", "hotel-shortname-2"),
			expectedNameConflictCondition: nameConflictCondition("PluralConflict", `"alfa" is already in use`),
			expectedEstablishedCondition:  notEstablishedCondition,
		},
		{
			name: "conflicting, not installing before with false condition",
			in: newGRD("alfa.bravo.com").SpecNames("alfa", "delta-singular", "echo-kind", "foxtrot-listkind", "golf-shortname-1", "hotel-shortname-2").
				Condition(notAcceptedCondition).
				NewOrDie(),
			existing: []*kindsv1.GrafanaResourceDefinition{
				newGRD("india.bravo.com").StatusNames("india", "alfa", "", "").NewOrDie(),
			},
			expectedNames:                 names("", "delta-singular", "echo-kind", "foxtrot-listkind", "golf-shortname-1", "hotel-shortname-2"),
			expectedNameConflictCondition: nameConflictCondition("PluralConflict", `"alfa" is already in use`),
			expectedEstablishedCondition:  notEstablishedCondition,
		},
	}

	for _, tc := range tests {
		grdIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
		for _, obj := range tc.existing {
			grdIndexer.Add(obj)
		}

		c := NamingConditionController{
			grdLister:        listers.NewGrafanaResourceDefinitionLister(grdIndexer),
			grdMutationCache: cache.NewIntegerResourceVersionMutationCache(grdIndexer, grdIndexer, 60*time.Second, false),
		}
		actualNames, actualNameConflictCondition, establishedCondition := c.calculateNamesAndConditions(tc.in)

		if e, a := tc.expectedNames, actualNames; !reflect.DeepEqual(e, a) {
			t.Errorf("%v expected %v, got %#v", tc.name, e, a)
		}
		if e, a := tc.expectedNameConflictCondition, actualNameConflictCondition; !kindshelpers.IsGRDConditionEquivalent(&e, &a) {
			t.Errorf("%v expected %v, got %v", tc.name, e, a)
		}
		if e, a := tc.expectedEstablishedCondition, establishedCondition; !kindshelpers.IsGRDConditionEquivalent(&e, &a) {
			t.Errorf("%v expected %v, got %v", tc.name, e, a)
		}
	}
}
