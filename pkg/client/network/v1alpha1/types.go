package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Prometheus defines a Prometheus deployment.
type FlannelNetwork struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              FlannelSpec    `json:"spec"`
}

type FlannelSpec struct {
	VNI string `json:"vni,omitempty"`
	Cidr string `json:"cidr,omitempty"`
}