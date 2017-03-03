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
	"k8s.io/client-go/1.5/pkg/api/resource"
	"k8s.io/client-go/1.5/pkg/api/unversioned"
	"k8s.io/client-go/1.5/pkg/api/v1"
	"k8s.io/client-go/1.5/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/1.5/pkg/runtime"
	"k8s.io/client-go/1.5/pkg/util/intstr"
	"k8s.io/client-go/1.5/pkg/watch"
	"k8s.io/client-go/1.5/rest"
	"k8s.io/client-go/1.5/tools/cache"
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

	fclient, err := v1alpha1.NewForConfig(cfg)
	if err != nil {
		log.Notice("Failed to get fclient: %v", err)
		return nil, err
	}

	o := &Operator{
		kclient: kclient,
		fclient: fclient,
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
	log.Notice("Creating DaemonSet for flannel-server")

	namespace := "kube-system"
	name := "flannel-server"
	version := "v0.6.2"

	dsetClient := c.kclient.ExtensionsClient.DaemonSets(namespace)

	// this is based on Timo's gist
	// https://gist.github.com/teemow/89dec8b5124123714f4036a76d7e74aa
	daemonSet := &v1beta1.DaemonSet{
		TypeMeta: unversioned.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "extensions/v1beta1",
		},
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels: map[string]string{
				"app":     name,
				"version": version,
			},
		},
		Spec: v1beta1.DaemonSetSpec{
			// Selector? We don't need the that here, I guess.
			//Selector: &unversioned.LabelSelector{
			//	MatchLabels: map[string]string{
			//		"app":		name,
			//		"version":	version,
			//	},
			//},
			Template: v1.PodTemplateSpec{
				// Do we need those? Won't harm, I guess..
				ObjectMeta: v1.ObjectMeta{
					Annotations: map[string]string{
						"scheduler.alpha.kubernetes.io/critical-pod": "",
						"scheduler.alpha.kubernetes.io/tolerations":  "[{\"key\":\"CriticalAddonsOnly\", \"operator\":\"Exists\"}]",
					},
					Labels: map[string]string{
						"app":   name,
						version: version,
					},
				},
				Spec: v1.PodSpec{
					HostNetwork: true,
					Containers: []v1.Container{
						{
							// Flannel running in server mode listens for connections on 8889
							Name:  "flannel-server",
							Image: "giantswarm/flannel:latest",
							Env: []v1.EnvVar{
								{
									Name: "HOST_PUBLIC_IP",
									ValueFrom: &v1.EnvVarSource{
										FieldRef: &v1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
							},
							Command: []string{
								"/bin/sh",
								"-c",
								"/opt/bin/flanneld -listen ${HOST_PUBLIC_IP}:8889 -etcd-endpoints http://${HOST_PUBLIC_IP}:2379 -ip-masq=true",
							},
							Ports: []v1.ContainerPort{
								{
									HostPort:      8889,
									ContainerPort: 8889,
								},
							},
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									"cpu": resource.Quantity{
										Format: "200m",
									},
								},
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "varlogflannel",
									MountPath: "/var/log",
								},
								{
									Name:      "varrunflannel",
									MountPath: "/var/run/flannel",
								},
							},
							LivenessProbe: &v1.Probe{
								Handler: v1.Handler{
									TCPSocket: &v1.TCPSocketAction{
										Port: intstr.IntOrString{
											IntVal: 8889,
										},
									},
								},
								InitialDelaySeconds: 30,
								TimeoutSeconds:      5,
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "varlogflannel",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/var/log/flannel",
								},
							},
						}, {
							Name: "varrunflannel",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/var/run/flannel",
								},
							},
						},
					},
				},
			},
		},
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
