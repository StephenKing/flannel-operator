// Copyright 2017 Steffen Gebert
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package flannel

import (
	"fmt"
	"time"

	"k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/api/v1"
	"k8s.io/client-go/1.5/pkg/api/unversioned"
	"k8s.io/client-go/1.5/pkg/runtime"
	"k8s.io/client-go/1.5/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/1.5/pkg/watch"
	"k8s.io/client-go/1.5/tools/cache"
	"k8s.io/client-go/1.5/rest"
)

const (
	resyncPeriod = 1 * time.Minute
)

// Operator manages the life cycle of the flannel deployments
type Operator struct {
	kclient *kubernetes.Clientset

	flanInf cache.SharedIndexInformer
	dsetInf cache.SharedIndexInformer
	nodeInf cache.SharedIndexInformer
}

// New creates a new controller
func New(cfg *rest.Config) (*Operator, error) {
	kclient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	c := &Operator{
		kclient: kclient,
	}


	// TODO not really sure if we need to watch for new DaemonSets.. we create one,
	// that should go to all nodes automatically. But let's see.. we probably need
	// such watch certainly for new FlannelNetwork creations to make sure that we
	// have a FlannelClient running.
	c.dsetInf = cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options api.ListOptions) (runtime.Object, error) {
				return kclient.DaemonSets(api.NamespaceAll).List(options)
			},
			WatchFunc: func(options api.ListOptions) (watch.Interface, error) {
				return kclient.DaemonSets(api.NamespaceAll).Watch(options)
			},
		},
		&v1beta1.DaemonSet{}, resyncPeriod, cache.Indexers{},
	)
	c.dsetInf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.handleAddDaemonSet,
		DeleteFunc: c.handleDeleteDaemonSet,
		UpdateFunc: c.handleUpdateDaemonSet,
	})

	c.createDaemonSet()

	return c, nil
}

func (c *Operator) createDaemonSet() error {
	namespace := "default"
	dsetClient := c.kclient.ExtensionsClient.DaemonSets(namespace)

	daemonSet := &v1beta1.DaemonSet{
		TypeMeta: unversioned.TypeMeta{
	 		Kind:       "DaemonSet",
			APIVersion: "extensions/v1beta1",
		},
		ObjectMeta: v1.ObjectMeta{
			Namespace:   "default",
			Name:        "flannel",
		},
		Spec: v1beta1.DaemonSetSpec{
			Template: v1.PodTemplateSpec{
				//MetaData: v1.ObjectMeta{
				//	Name: "flannel"
				//},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name: "flannel-server",
							Image: "giantswarm/flannel:latest",
						},
					},
				},
			},
		},
		// Status: extensions.DaemonSetStatus{},
	}


	if _, err := dsetClient.Create(daemonSet); err != nil {
		return fmt.Errorf("create daemonset: %s", err)
	}
	return nil
}


func (c *Operator) handleAddDaemonSet(obj interface{}) {
	// TODO
	//if flanSet := c.flannelForDaemonSet(obj); flanSet != nil {
	//	c.enqueue(flanSet)
	//}
}
func (c *Operator) handleDeleteDaemonSet(obj interface{}) {
	// TODO
	//if flanSet := c.flannelForDaemonSet(obj); flanSet != nil {
	//	c.enqueue(flanSet)
	//}
}

func (c *Operator) handleUpdateDaemonSet(oldo, curo interface{}) {
	// TODO
	//old := oldo.(*extensions.DaemonSet)
	//cur := oldo.(*extensions.DaemonSet)
	//
	//c.logger.Log("msg", "update handler", "old", old.ResourceVersion, "cur", cur.ResourceVersion)
	//
	//// Periodic resync may resend the deployment without changes in-between.
	//// Also breaks loops created by updating the resource ourselves.
	//if old.ResourceVersion == cur.ResourceVersion {
	//	return
	//}
	//
	//if flanSet := c.flannelForDaemonSet(cur); flanSet != nil {
	//	c.enqueue(flanSet)
	//}
}