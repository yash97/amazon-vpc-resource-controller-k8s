// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package node

import (
	"context"
	"strings"

	cninode "github.com/aws/amazon-vpc-resource-controller-k8s/apis/vpcresources/v1alpha1"
	"github.com/aws/amazon-vpc-resource-controller-k8s/test/framework/utils"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Manager interface {
	GetNodesWithOS(os string) (*v1.NodeList, error)
	AddLabels(nodeList []v1.Node, label map[string]string) error
	RemoveLabels(nodeList []v1.Node, label map[string]string) error
	GetNode(node *v1.Node) (*v1.Node, error)
	GetNodeList() (*v1.NodeList, error)
	GetCNINode(node *v1.Node) (*cninode.CNINode, error)
	GetCNINodeList() (*cninode.CNINodeList, error)
	GetInstanceID(node *v1.Node) string
}

type defaultManager struct {
	k8sClient client.Client
}

func NewManager(k8sClient client.Client) Manager {
	return &defaultManager{k8sClient: k8sClient}
}

func (d *defaultManager) GetNodesWithOS(os string) (*v1.NodeList, error) {
	nodeList := &v1.NodeList{}
	err := d.k8sClient.List(context.TODO(), nodeList, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{"kubernetes.io/os": os}),
	})
	return nodeList, err
}

func (d *defaultManager) AddLabels(nodeList []v1.Node, label map[string]string) error {
	for _, node := range nodeList {
		latestNode := &v1.Node{}
		// Get the latest node object reference from cache
		err := d.k8sClient.Get(context.TODO(), types.NamespacedName{
			Name: node.Name,
		}, latestNode)
		if err != nil {
			return err
		}
		patchedNode := latestNode.DeepCopy()
		for labelKey, labelVal := range label {
			patchedNode.Labels[labelKey] = labelVal
		}
		err = d.k8sClient.Patch(context.TODO(), patchedNode, client.MergeFrom(latestNode))
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *defaultManager) RemoveLabels(nodeList []v1.Node, label map[string]string) error {
	for _, node := range nodeList {
		latestNode := &v1.Node{}
		// Get the latest node object reference from cache
		err := d.k8sClient.Get(context.TODO(), types.NamespacedName{
			Name: node.Name,
		}, latestNode)
		if err != nil {
			return err
		}
		patchedNode := latestNode.DeepCopy()
		for labelKey, _ := range label {
			delete(patchedNode.Labels, labelKey)
		}
		err = d.k8sClient.Patch(context.TODO(), patchedNode, client.MergeFrom(latestNode))
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *defaultManager) GetNode(node *v1.Node) (*v1.Node, error) {
	observedNode := &v1.Node{}
	err := d.k8sClient.Get(context.TODO(), utils.NamespacedName(node), observedNode)
	return observedNode, err
}

func (d *defaultManager) GetCNINode(node *v1.Node) (*cninode.CNINode, error) {
	cniNode := &cninode.CNINode{}
	err := d.k8sClient.Get(context.TODO(), types.NamespacedName{Name: node.Name}, cniNode)
	return cniNode, err
}

func (d *defaultManager) GetCNINodeList() (*cninode.CNINodeList, error) {
	list := &cninode.CNINodeList{}
	err := d.k8sClient.List(context.TODO(), list)
	return list, err
}

func (d *defaultManager) GetNodeList() (*v1.NodeList, error) {
	list := &v1.NodeList{}
	err := d.k8sClient.List(context.TODO(), list)
	return list, err
}

func (d *defaultManager) GetInstanceID(node *v1.Node) string {
	if node.Spec.ProviderID != "" {
		id := strings.Split(node.Spec.ProviderID, "/")
		return id[len(id)-1]
	}
	return ""
}
