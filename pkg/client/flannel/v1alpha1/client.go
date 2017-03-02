package v1alpha1

import (
	"k8s.io/client-go/1.5/rest"
	"k8s.io/client-go/1.5/dynamic"
)
const (
	TPRGroup   = "flannel.st-g.de"
	TPRVersion = "v1alpha1"
)

type FlannelV1alpha1Interface interface {
	RESTClient()  *rest.RESTClient
	FlannelsGetter
}

type FlannelV1alpha1Client struct {
	restClient    *rest.RESTClient
	dynamicClient *dynamic.Client
}

func (c *FlannelV1alpha1Client) Flannels(namespace string) FlannelInterface {
	return newFlannels(c.restClient, c.dynamicClient, namespace)
}

func (c *FlannelV1alpha1Client) RESTClient() *rest.RESTClient {
	return c.restClient
}