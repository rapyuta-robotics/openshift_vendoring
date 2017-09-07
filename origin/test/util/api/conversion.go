package api

import (
	"testing"

	kapi "github.com/openshift/kubernetes/pkg/api"
)

func CheckFieldLabelConversions(t *testing.T, version, kind string, expectedLabels map[string]string, customLabels ...string) {
	for label := range expectedLabels {
		_, _, err := kapi.Scheme.ConvertFieldLabel(version, kind, label, "")
		if err != nil {
			t.Errorf("No conversion registered for %s for %s %s", label, version, kind)
		}
	}
	for _, label := range customLabels {
		_, _, err := kapi.Scheme.ConvertFieldLabel(version, kind, label, "")
		if err != nil {
			t.Errorf("No conversion registered for %s for %s %s", label, version, kind)
		}
	}
}
