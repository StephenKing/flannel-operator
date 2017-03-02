package v1alpha1

import (
	"k8s.io/client-go/1.5/rest"
	"k8s.io/client-go/1.5/dynamic"
	"k8s.io/client-go/1.5/pkg/api/unversioned"
	"k8s.io/client-go/1.5/pkg/runtime"
	"k8s.io/client-go/1.5/pkg/runtime/serializer"
	"k8s.io/client-go/1.5/pkg/api"
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

func NewForConfig(c *rest.Config) (*FlannelNetworkV1alpha1Client, error) {
	config := *c
	setConfigDefaults(&config)
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}

	dynamicClient, err := dynamic.NewClient(&config)
	if err != nil {
		return nil, err
	}

	return &FlannelNetworkV1alpha1Client{client, dynamicClient}, nil
}

func setConfigDefaults(config *rest.Config) {
	config.GroupVersion = &unversioned.GroupVersion{
		Group:   TPRGroup,
		Version: TPRVersion,
	}
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: api.Codecs}
}
