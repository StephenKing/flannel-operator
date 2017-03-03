package v1alpha1

import (
	"k8s.io/client-go/1.5/dynamic"
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/api/unversioned"
	"k8s.io/client-go/1.5/pkg/runtime"
	"k8s.io/client-go/1.5/pkg/runtime/serializer"
	"k8s.io/client-go/1.5/rest"
)

const (
	TPRGroup   = "flannel.st-g.de"
	TPRVersion = "v1alpha1"
)

type FlannelNetworkV1alpha1Interface interface {
	RESTClient() *rest.RESTClient
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
	log.Notice("Creating new FlannelNetworkV1alpha1Client")
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

	log.Notice("Finished creating FlannelNetworkV1alpha1Client")
	return &FlannelNetworkV1alpha1Client{client, dynamicClient}, nil
}

func setConfigDefaults(config *rest.Config) {
	log.Notice("Setting up REST default configs")
	config.GroupVersion = &unversioned.GroupVersion{
		Group:   TPRGroup,
		Version: TPRVersion,
	}
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: api.Codecs}
}
