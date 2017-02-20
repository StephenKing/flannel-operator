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

	// "github.com/StephenKing/flannel-operator/pkg/client/network/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/go-kit/kit/log"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/client-go/rest"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"
	"net/url"
)

const (
	resyncPeriod = 1 * time.Minute
)

// Operator manages the life cycle of the flannel deployments
type Operator struct {
	kclient *kubernetes.Clientset
	logger log.Logger

	flanInf cache.SharedIndexInformer
	dsetInf cache.SharedIndexInformer
	nodeInf cache.SharedIndexInformer

	queue workqueue.RateLimitingInterface

	host                   string
}

// Config defines configuration parameters for the Operator.
type Config struct {
	Host          string
	KubeletObject string
	TLSInsecure   bool
	TLSConfig     rest.TLSClientConfig
}

// New creates a new controller
func New(conf Config, logger log.Logger) (*Operator, error) {
	cfg, err := NewClusterConfig(conf.Host, conf.TLSInsecure, &conf.TLSConfig)
	if err != nil {
		return nil, err
	}
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	c := &Operator{
		kclient:                client,
		logger:                 logger,
		queue:                  workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "flannel"),
		host:                   conf.Host,
	}


	c.createDaemonSet()

	// TODO add event handler for CRUD operations, esp. one for the DaemonSet

	//c.dsetInf = cache.NewSharedIndexInformer(
	//	cache.NewListWatchFromClient(c.kclient.Apps().RESTClient(), "daemonsets", api.NamespaceAll, nil),
	//	&v1beta1.StatefulSet{}, resyncPeriod, cache.Indexers{},
	//)
	//c.dsetInf.AddEventHandler(cache.ResourceEventHandlerFuncs{
	//	AddFunc:    c.handleAddDaemonSet,
	//	DeleteFunc: c.handleDeleteDaemonSet,
	//	UpdateFunc: c.handleUpdateDaemonSet,
	//})
	//
	//
	//if kubeletSyncEnabled {
	//	c.nodeInf = cache.NewSharedIndexInformer(
	//		cache.NewListWatchFromClient(c.kclient.Core().RESTClient(), "nodes", api.NamespaceAll, nil),
	//		&v1.Node{}, resyncPeriod, cache.Indexers{},
	//	)
	//	c.nodeInf.AddEventHandler(cache.ResourceEventHandlerFuncs{
	//		AddFunc:    c.handleAddNode,
	//		DeleteFunc: c.handleDeleteNode,
	//		UpdateFunc: c.handleUpdateNode,
	//	})
	//}


	return c, nil
}

func (c *Operator) createDaemonSet() error {
	namespace := "default"
	dsetClient := c.kclient.ExtensionsV1beta1Client.DaemonSets(namespace)

	daemonSet := &extensions.DaemonSet{
		//TypeMeta: metav1.TypeMeta{
	 	//	Kind:       "DaemonSet",
		//	APIVersion: "extensions/v1beta1",
		//},
		// TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   "default",
			Name:        "flannel",
		},
		Spec: extensions.DaemonSetSpec{
			// Template:
		},
		// Status: extensions.DaemonSetStatus{},
	}

	if _, err := dsetClient.Create(daemonSet); err != nil {
		return fmt.Errorf("create daemonset: %s", err)
	}
	return nil
}


func NewClusterConfig(host string, tlsInsecure bool, tlsConfig *rest.TLSClientConfig) (*rest.Config, error) {
	var cfg *rest.Config
	var err error

	if len(host) == 0 {
		if cfg, err = rest.InClusterConfig(); err != nil {
			return nil, err
		}
	} else {
		cfg = &rest.Config{
			Host: host,
		}
		hostURL, err := url.Parse(host)
		if err != nil {
			return nil, fmt.Errorf("error parsing host url %s : %v", host, err)
		}
		if hostURL.Scheme == "https" {
			cfg.TLSClientConfig = *tlsConfig
			cfg.Insecure = tlsInsecure
		}
	}
	cfg.QPS = 100
	cfg.Burst = 100

	return cfg, nil
}

//func (c *Operator) keyFunc(obj interface{}) (string, bool) {
//	k, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
//	if err != nil {
//		c.logger.Log("msg", "creating key failed", "err", err)
//		return k, false
//	}
//	return k, true
//}
//
//// enqueue adds a key to the queue. If obj is a key already it gets added directly.
//// Otherwise, the key is extracted via keyFunc.
//func (c *Operator) enqueue(obj interface{}) {
//	if obj == nil {
//		return
//	}
//
//	key, ok := obj.(string)
//	if !ok {
//		key, ok = c.keyFunc(obj)
//		if !ok {
//			return
//		}
//	}
//
//	c.queue.Add(key)
//}
//
//func daemonSetKeytoFlannelKey(key string) string {
//	keyParts := strings.Split(key, "/")
//	return keyParts[0] + "/" + strings.TrimPrefix(keyParts[1], "flannel-")
//}
//
//func flannelKeyToDaemonSetKey(key string) string {
//	keyParts := strings.Split(key, "/")
//	return keyParts[0] + "/flannel-" + keyParts[1]
//}
//
//func (c *Operator) flannelForDaemonSet(dset interface{}) *v1alpha1.FlannelNetwork {
//	key, ok := c.keyFunc(dset)
//	if !ok {
//		return nil
//	}
//
//	flanKey := daemonSetKeytoFlannelKey(key)
//	f, exists, err := c.flanInf.GetStore().GetByKey(flanKey)
//	if err != nil {
//		c.logger.Log("msg", "Flannel lookup failed", "err", err)
//		return nil
//	}
//	if !exists {
//		return nil
//	}
//	return f.(*v1alpha1.FlannelNetwork)
//}
//
//func (c *Operator) handleAddDaemonSet(obj interface{}) {
//	if flanSet := c.flannelForDaemonSet(obj); flanSet != nil {
//		c.enqueue(flanSet)
//	}
//}
//func (c *Operator) handleDeleteDaemonSet(obj interface{}) {
//	if flanSet := c.flannelForDaemonSet(obj); flanSet != nil {
//		c.enqueue(flanSet)
//	}
//}
//
//func (c *Operator) handleUpdateDaemonSet(oldo, curo interface{}) {
//	old := oldo.(*extensions.DaemonSet)
//	cur := oldo.(*extensions.DaemonSet)
//
//	c.logger.Log("msg", "update handler", "old", old.ResourceVersion, "cur", cur.ResourceVersion)
//
//	// Periodic resync may resend the deployment without changes in-between.
//	// Also breaks loops created by updating the resource ourselves.
//	if old.ResourceVersion == cur.ResourceVersion {
//		return
//	}
//
//	if flanSet := c.flannelForDaemonSet(cur); flanSet != nil {
//		c.enqueue(flanSet)
//	}
//}

//func (c *Operator) handleAddNode(obj interface{})         { c.syncNodeEndpoints() }
//func (c *Operator) handleDeleteNode(obj interface{})      { c.syncNodeEndpoints() }
//func (c *Operator) handleUpdateNode(old, cur interface{}) { c.syncNodeEndpoints() }
//
//func (c *Operator) syncNodeEndpoints() error {
//
//
//	namespace := "default"
//	dsetClient := c.kclient.ExtensionsV1beta1Client.DaemonSets(namespace)
//
//	// Ensure we have a DaemonSet running flannel-server deployed.
//	obj, exists, err = c.dsetInf.GetIndexer().GetByKey(flannelKeyToDaemonSetKey(key))
//
//	daemonset := extensions.DaemonSet{
//		// TypeMeta: metav1.TypeMeta{},
//		ObjectMeta: metav1.ObjectMeta{Name: "foo"},
//		Spec: extensions.DaemonSetSpec{},
//		// Status: extensions.DaemonSetStatus{},
//	}
//
//	if _, err := dsetClient.Create(daemonset); err != nil {
//		return fmt.Errorf("create daemonset: %s", err)
//	}
//
//	/*
//	extensions.DaemonSet{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:        "flannel-server"
//		},
//		//	//Labels:      map[string]string{
//		//	//	"app:" "flannel-server",
//		//	//}
//		//	//Annotations: p.ObjectMeta.Annotations,
//		//},
//		Spec: extensions.DaemonSetSpec{
//			Selector: &metav1.LabelSelector{
//				MatchLabels: map[string]string{
//					"foo": "bar"}
//			}
//		}
//	}
//	*/
//
//}