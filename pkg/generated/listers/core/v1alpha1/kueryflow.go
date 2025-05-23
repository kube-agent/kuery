/*
Copyright The Kuery Authors.

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

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"

	v1alpha1 "github.com/kube-agent/kuery/api/core/v1alpha1"
)

// KueryFlowLister helps list KueryFlows.
// All objects returned here must be treated as read-only.
type KueryFlowLister interface {
	// List lists all KueryFlows in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.KueryFlow, err error)
	// KueryFlows returns an object that can list and get KueryFlows.
	KueryFlows(namespace string) KueryFlowNamespaceLister
	KueryFlowListerExpansion
}

// kueryFlowLister implements the KueryFlowLister interface.
type kueryFlowLister struct {
	indexer cache.Indexer
}

// NewKueryFlowLister returns a new KueryFlowLister.
func NewKueryFlowLister(indexer cache.Indexer) KueryFlowLister {
	return &kueryFlowLister{indexer: indexer}
}

// List lists all KueryFlows in the indexer.
func (s *kueryFlowLister) List(selector labels.Selector) (ret []*v1alpha1.KueryFlow, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.KueryFlow))
	})
	return ret, err
}

// KueryFlows returns an object that can list and get KueryFlows.
func (s *kueryFlowLister) KueryFlows(namespace string) KueryFlowNamespaceLister {
	return kueryFlowNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// KueryFlowNamespaceLister helps list and get KueryFlows.
// All objects returned here must be treated as read-only.
type KueryFlowNamespaceLister interface {
	// List lists all KueryFlows in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.KueryFlow, err error)
	// Get retrieves the KueryFlow from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.KueryFlow, error)
	KueryFlowNamespaceListerExpansion
}

// kueryFlowNamespaceLister implements the KueryFlowNamespaceLister
// interface.
type kueryFlowNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all KueryFlows in the indexer for a given namespace.
func (s kueryFlowNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.KueryFlow, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.KueryFlow))
	})
	return ret, err
}

// Get retrieves the KueryFlow from the indexer for a given namespace and name.
func (s kueryFlowNamespaceLister) Get(name string) (*v1alpha1.KueryFlow, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("kueryflow"), name)
	}
	return obj.(*v1alpha1.KueryFlow), nil
}
