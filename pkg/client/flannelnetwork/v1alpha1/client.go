package v1alpha1

import (
	"k8s.io/client-go/1.5/rest"
	"k8s.io/client-go/1.5/dynamic"
)
const (
	TPRGroup   = "flannel.st-g.de"
	TPRVersion = "v1alpha1"
)

type FlannelNetworkV1alpha1Interface interface {
	RESTClient()  *rest.RESTClient
	FlannelNetworksGetter
}

type FlannelNetworkV1alpha1Client struct {
	restClient    *rest.RESTClient
	dynamicClient *dynamic.Client
}

func (c *FlannelNetworkV1alpha1Client) FlannelNetworks(namespace string) FlannelNetworkInterface {
	return newFlannelNetworks(c.restClient, c.dynamicClient, namespace)
}

func (c *FlannelNetworkV1alpha1Client) RESTClient() *rest.RESTClient {
	return c.restClient
}