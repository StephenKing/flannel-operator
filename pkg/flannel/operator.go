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
	"k8s.io/client-go/1.5/pkg/api/errors"
	"k8s.io/client-go/1.5/pkg/api/resource"
	"k8s.io/client-go/1.5/pkg/api/unversioned"
	"k8s.io/client-go/1.5/pkg/api/v1"
	"k8s.io/client-go/1.5/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/1.5/pkg/util/intstr"
	"k8s.io/client-go/1.5/rest"
	"k8s.io/client-go/1.5/tools/cache"
)

var (
	log = logging.MustGetLogger("flannel-operator")
)

const (
	resyncPeriod = 1 * time.Minute
	kubeSystemNamespace = "kube-system"
	dsetFlannelName = "flannel-server"
	dsetFlannelVersion = "v0.6.2"
	tprFlannelNetwork = "flannel-network." + v1alpha1.TPRGroup
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
			ListFunc: o.fclient.FlannelNetworks(api.NamespaceAll).List,
			WatchFunc: o.fclient.FlannelNetworks(api.NamespaceAll).Watch,
		},
		&v1alpha1.FlannelNetwork{}, resyncPeriod, cache.Indexers{},
	)
	o.flanInf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    o.handleAddFlannelNetwork,
		DeleteFunc: o.handleDeleteFlannelNetwork,
		//UpdateFunc: o.handleUpdateFlannelNetwork,
	})

	log.Notice("Added Event handlers")

	o.createDaemonSet()

	log.Notice("Done with Operator.New")

	return o, nil
}

func (c *Operator) Run(stopc <-chan struct{}) error {
	log.Notice("Called Operator.Run")
	go c.flanInf.Run(stopc)

	defer c.Stop()

	if err := c.createTPRs(); err != nil {
		log.Warning("Create TPRs failed:", err)
	}

	<-stopc
	log.Notice("Operator.Run received stop signal")
	return nil
}

func (c *Operator) Stop() error {
	log.Notice("Shutting down operator")


	if err := c.deleteTPRs(); err != nil {
		log.Error("Deleting TPR failed:", err)
	}

	log.Notice("Leaving all FlannelNetworks in place")
	log.Notice("Leaving flannel-client deployments in place")
	log.Notice("Leaving flannel-server DaemonSets in place")

	return nil
}

func (c *Operator) createDaemonSet() error {
	log.Notice("Creating DaemonSet for flannel-server")

	dsetClient := c.kclient.ExtensionsClient.DaemonSets(kubeSystemNamespace)

	// this is based on Timo's gist
	// https://gist.github.com/teemow/89dec8b5124123714f4036a76d7e74aa
	daemonSet := &v1beta1.DaemonSet{
		TypeMeta: unversioned.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "extensions/v1beta1",
		},
		ObjectMeta: v1.ObjectMeta{
			Namespace: kubeSystemNamespace,
			Name:      dsetFlannelName,
			Labels: map[string]string{
				"app":     dsetFlannelName,
				"version": dsetFlannelVersion,
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
						"app":     dsetFlannelName,
						"version": dsetFlannelVersion,
					},
				},
				Spec: v1.PodSpec{
					HostNetwork: true,
					Containers: []v1.Container{
						{
							// Flannel running in server mode listens for connections on 8889
							Name:  "flannel-server",
							Image: "giantswarm/flannel:" + dsetFlannelVersion,
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

	log.Notice("DaemonSet created")
	return nil
}

func (c *Operator) deleteDaemonSet() error {
	log.Notice("Deleting DaemonSet", dsetFlannelName)

	dsetClient := c.kclient.ExtensionsClient.DaemonSets(kubeSystemNamespace)
	// remove all the pods, not only the DaemonSet
	var orphan bool = true
	deleteOptions := &api.DeleteOptions{
		OrphanDependents: &orphan,
	}

	return dsetClient.Delete(dsetFlannelName, deleteOptions)
}

func (c *Operator) createTPRs() error {

	tpr := &v1beta1.ThirdPartyResource{
		ObjectMeta: v1.ObjectMeta{
			Name: tprFlannelNetwork,
			// TODO some labels could help?
			// Labels:
		},
		Versions: []v1beta1.APIVersion{
			{Name: v1alpha1.TPRVersion},
		},
		Description: "Flannel-based container network connectivity",
	}

	tprClient := c.kclient.ExtensionsClient.ThirdPartyResources()

	if _, err := tprClient.Create(tpr); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	log.Notice("TPR created:", tpr.Name)
	return nil
}

func (c *Operator) deleteTPRs() error {
	log.Notice("Deleting TPR", tprFlannelNetwork)

	tprClient := c.kclient.ExtensionsClient.ThirdPartyResources()

	return tprClient.Delete(tprFlannelNetwork, &api.DeleteOptions{})
}

func (c *Operator) handleAddFlannelNetwork(obj interface{}) {

	flan := obj.(*v1alpha1.FlannelNetwork)
	vni := flan.Spec.VNI
	cidr := flan.Spec.Cidr

	log.Notice("FlannelNetwork added (ns", flan.Namespace, " | VNI", vni, "| CIDR", cidr, ")")
	log.Notice("Creating deployment of new flannel client")

	var replicas int32 = 1
	var privileged bool = true

	depl := &v1beta1.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name: "flannel-client-" + flan.Namespace + "-" + flan.Name + "-vni" + vni,
			Labels: map[string]string{
				"app": "flannel-client",
				"vni": vni,
			},
			Namespace: kubeSystemNamespace,
		},
		Spec: v1beta1.DeploymentSpec{
			Strategy: v1beta1.DeploymentStrategy{
				Type: "Recreate",
			},
			Replicas: &replicas,
			Template: v1.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Name: clientDeploymentName(flan),
					Labels: map[string]string{
						"app": "flannel-client",
						"vni": vni,
					},
					Annotations: map[string]string{
						"seccomp.security.alpha.kubernetes.io/pod": "unconfined",
					},
				},
				Spec: v1.PodSpec{
					HostNetwork: true,
					Volumes: []v1.Volume{
						{
							Name: "flannel",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/run/flannel",
								},
							},
						},
					},
					RestartPolicy: "Always",
					Containers: []v1.Container{
						{
							Name: "k8s-flannel",
							SecurityContext: &v1.SecurityContext{
								Privileged: &privileged,
							},
							Image:           "giantswarm/flannel:v0.6.2",
							ImagePullPolicy: "IfNotPresent",
							Env: []v1.EnvVar{
								{
									Name: "NODE_IP",
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
								"/opt/bin/flanneld --remote=$NODE_IP:8889 --public-ip=$NODE_IP --iface=$NODE_IP --networks=" + vni + " -v=1",
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "flannel",
									MountPath: "/run/flannel",
								},
							},
						},
					},
				},
			},
		},
	}

	deplClient := c.kclient.Deployments(kubeSystemNamespace)
	if _, err := deplClient.Create(depl); err != nil {
		log.Error("Creating deployment failed:", err)
	} else {
		log.Notice("Deployment for flannel client created")
	}

}


func (c *Operator) handleDeleteFlannelNetwork(obj interface{}) {
	flan := obj.(*v1alpha1.FlannelNetwork)
	vni := flan.Spec.VNI
	cidr := flan.Spec.Cidr

	log.Notice("handleDeleteFlannelNetwork (VNI ", vni, ", CIDR", cidr, ")")
	deploymentClient := c.kclient.Deployments(kubeSystemNamespace)

	// remove all the pods, not only the Deployment
	var orphan bool = true
	deleteOptions := &api.DeleteOptions{
		OrphanDependents: &orphan,
	}

	if err := deploymentClient.Delete(clientDeploymentName(flan), deleteOptions); err != nil {
		log.Error("Deleting deployment flannel-client failed:", err)
	} else {
		log.Notice("Deleted deployment flannel-client")
	}
}

func clientDeploymentName(flan *v1alpha1.FlannelNetwork) string {
	return "flannel-client-" + flan.Namespace + "-" + flan.Name + "-vni" + flan.Spec.VNI
}
