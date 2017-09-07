package node

import (
	"fmt"

	"github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/kubectl"
	kerrors "github.com/openshift/kubernetes/pkg/util/errors"
)

type SchedulableOptions struct {
	Options *NodeOptions

	Schedulable bool
}

func (s *SchedulableOptions) Run() error {
	nodes, err := s.Options.GetNodes()
	if err != nil {
		return err
	}

	errList := []error{}
	var printer kubectl.ResourcePrinter
	unschedulable := !s.Schedulable
	for _, node := range nodes {
		if node.Spec.Unschedulable != unschedulable {
			patch := fmt.Sprintf(`{"spec":{"unschedulable":%t}}`, unschedulable)
			node, err = s.Options.KubeClient.Core().Nodes().Patch(node.Name, api.MergePatchType, []byte(patch))
			if err != nil {
				errList = append(errList, err)
				continue
			}
		}

		if printer == nil {
			p, err := s.Options.GetPrintersByObject(node)
			if err != nil {
				return err
			}
			printer = p
		}

		printer.PrintObj(node, s.Options.Writer)
	}
	return kerrors.NewAggregate(errList)
}
