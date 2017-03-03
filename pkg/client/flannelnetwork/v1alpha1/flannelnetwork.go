package v1alpha1

import (
	"encoding/json"

	"github.com/op/go-logging"

	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/api/v1"

	// why not? rest.Interface is missing
	// "k8s.io/client-go/1.5/rest"
	"k8s.io/client-go/1.5/rest"

	"k8s.io/client-go/1.5/dynamic"
	"k8s.io/client-go/1.5/pkg/runtime"
	"k8s.io/client-go/1.5/pkg/watch"
	// "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/1.5/pkg/api/unversioned"
)

var (
	log = logging.MustGetLogger("flannel-network")
)

const (
	TPRFlannelKind = "FlannelNetwork"
	TPRFlannelName = "flannelnetworks"
)

type FlannelNetworksGetter interface {
	Flannels(namespace string) FlannelNetworkInterface
}

type FlannelNetworkInterface interface {
	Create(*FlannelNetwork) (*FlannelNetwork, error)
	Get(name string) (*FlannelNetwork, error)
	Update(*FlannelNetwork) (*FlannelNetwork, error)
	Delete(name string, options *v1.DeleteOptions)
	List(opts api.ListOptions) (runtime.Object, error)
	Watch(opts api.ListOptions) (watch.Interface, error)
}

type flannelnetworks struct {
	restClient *rest.RESTClient
	client     *dynamic.ResourceClient
	ns         string
}

func newFlannelNetworks(r *rest.RESTClient, c *dynamic.Client, namespace string) *flannelnetworks {
	log.Notice("Called newFlannelNetworks")

	return &flannelnetworks{
		r,
		c.Resource(
			&unversioned.APIResource{
				Kind:       TPRFlannelKind,
				Name:       TPRFlannelName,
				Namespaced: true,
			},
			namespace,
		),
		namespace,
	}
}

func (f *flannelnetworks) Create(o *FlannelNetwork) (*FlannelNetwork, error) {
	log.Notice("*flannelnetworks.Create")

	up, err := UnstructuredFromFlannelNetwork(o)
	if err != nil {
		return nil, err
	}

	up, err = f.client.Create(up)
	if err != nil {
		return nil, err
	}

	return FlannelNetworkFromUnstructured(up)
}

func (f *flannelnetworks) Get(name string) (*FlannelNetwork, error) {
	log.Notice("*flannelnetworks.Get")

	obj, err := f.client.Get(name)
	if err != nil {
		return nil, err
	}
	return FlannelNetworkFromUnstructured(obj)
}

func (f *flannelnetworks) Update(o *FlannelNetwork) (*FlannelNetwork, error) {
	log.Notice("*flannelnetworks.Create")

	up, err := UnstructuredFromFlannelNetwork(o)
	if err != nil {
		return nil, err
	}

	up, err = f.client.Update(up)
	if err != nil {
		return nil, err
	}

	return FlannelNetworkFromUnstructured(up)
}

// TODO had to remove the return type "error" because of
// pkg/client/flannel/v1alpha1/client.go:25: cannot use newFlannelNetworks(c.restClient, c.dynamicClient, namespace) (type *flannelnetworks) as type FlannelNetworkInterface in return argument:
func (f *flannelnetworks) Delete(name string, options *v1.DeleteOptions) {
	log.Notice("*flannelnetworks.Delete")

	if err := f.client.Delete(name, options); err != nil {
		log.Error("Could not delete %v - %v", name, err)
	}
}

func (f *flannelnetworks) List(opts api.ListOptions) (runtime.Object, error) {
	log.Notice("*flannelnetworks.List")

	req := f.restClient.Get().
		Namespace(f.ns).
		Resource("flannelnetworks").
		FieldsSelectorParam(nil)

	b, err := req.DoRaw()
	if err != nil {
		log.Notice("*flannelnetworks.List did not work out: %v", err)
		return nil, err
	}
	var flan FlannelNetworkList
	log.Notice("*flannelnetworks.List finished")
	return &flan, json.Unmarshal(b, &flan)
}

func (f *flannelnetworks) Watch(opts api.ListOptions) (watch.Interface, error) {
	log.Notice("*flannelnetworks.Watch")

	r, err := f.restClient.Get().
		Prefix("watch").
		Namespace(f.ns).
		Resource("flannelnetworks").
		// VersionedParams(&options, v1.ParameterCodec).
		FieldsSelectorParam(nil).
		Stream()
	if err != nil {
		return nil, err
	}
	return watch.NewStreamWatcher(&flannelNetworkDecoder{
		dec:   json.NewDecoder(r),
		close: r.Close,
	}), nil
}

// FlannelNetworkFromUnstructured unmarshals a FlannelNetwork object from dynamic client's unstructured
func FlannelNetworkFromUnstructured(r *runtime.Unstructured) (*FlannelNetwork, error) {
	b, err := json.Marshal(r.Object)
	if err != nil {
		return nil, err
	}
	var f FlannelNetwork
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, err
	}
	f.TypeMeta.Kind = TPRFlannelKind
	f.TypeMeta.APIVersion = TPRGroup + "/" + TPRVersion
	return &f, nil
}

// UnstructuredFromFlannelNetwork marshals a FlannelNetwork object into dynamic client's unstructured
func UnstructuredFromFlannelNetwork(f *FlannelNetwork) (*runtime.Unstructured, error) {
	f.TypeMeta.Kind = TPRFlannelKind
	f.TypeMeta.APIVersion = TPRGroup + "/" + TPRVersion
	b, err := json.Marshal(f)
	if err != nil {
		return nil, err
	}
	var r runtime.Unstructured
	if err := json.Unmarshal(b, &r.Object); err != nil {
		return nil, err
	}
	return &r, nil
}

type flannelNetworkDecoder struct {
	dec   *json.Decoder
	close func() error
}

func (d *flannelNetworkDecoder) Close() {
	d.close()
}

func (d *flannelNetworkDecoder) Decode() (action watch.EventType, object runtime.Object, err error) {
	var e struct {
		Type   watch.EventType
		Object FlannelNetwork
	}
	if err := d.dec.Decode(&e); err != nil {
		return watch.Error, nil, err
	}
	return e.Type, &e.Object, nil
}
