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

	"github.com/op/go-logging"

	"github.com/StephenKing/flannel-operator/pkg/client/flannelnetwork/v1alpha1"

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

var (
	log = logging.MustGetLogger("flannel-operator")
)

const (
	resyncPeriod = 1 * time.Minute
)

// Operator manages the life cycle of the flannel deployments
type Operator struct {
	kclient *kubernetes.Clientset
	fclient *v1alpha1.FlannelNetworkV1alpha1Client

	flanInf cache.SharedIndexInformer
	nodeInf cache.SharedIndexInformer
}

// New creates a new controller
func New(cfg *rest.Config) (*Operator, error) {
	log.Notice("About to create new flannel operator")

	kclient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Notice("Failed to get kclient: %v", err)
		return nil, err
	}

	o := &Operator{
		kclient: kclient,
	}

	// Watch for new FlannelNetwork creations to make sure that we
	// have a FlannelClient running.
	o.flanInf = cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options api.ListOptions) (runtime.Object, error) {
				return o.fclient.FlannelNetworks(api.NamespaceAll).List(options)
			},
			WatchFunc: func(options api.ListOptions) (watch.Interface, error) {
				return o.fclient.FlannelNetworks(api.NamespaceAll).Watch(options)
			},
		},
		&v1alpha1.FlannelNetwork{}, resyncPeriod, cache.Indexers{},
	)
	o.flanInf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    o.handleAddFlannelNetwork,
		DeleteFunc: o.handleDeleteFlannelNetwork,
		// UpdateFunc: o.handleUpdateFlannelNetwork,
	})

	log.Notice("Added Event handlers")

	o.createDaemonSet()

	log.Notice("Done with Operator.New")

	return o, nil
}

func (c *Operator) Run(stopc <-chan struct{}) error {
	log.Notice("Called Operator.Run")
	go c.flanInf.Run(stopc)

	<-stopc
	log.Notice("Operator.Run received stop signal")
	return nil
}

func (c *Operator) createDaemonSet() error {
	log.Notice("Creating DaemonSet for flannel")

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

	log.Notice("DaemonSet seems created")
	return nil
}


func (c *Operator) handleAddFlannelNetwork(obj interface{}) {
	log.Warning("TODO: implement handleAddFlannelNetwork")
	// TODO
	//if flanSet := c.flannelForDaemonSet(obj); flanSet != nil {
	//	c.enqueue(flanSet)
	//}
}
func (c *Operator) handleDeleteFlannelNetwork(obj interface{}) {
	log.Warning("TODO: implement handleDeleteFlannelNetwork")
	// TODO
	//if flanSet := c.flannelForDaemonSet(obj); flanSet != nil {
	//	c.enqueue(flanSet)
	//}
}

// func (c *Operator) handleUpdateFlannelNetwork(oldo, curo interface{}) {
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
// }