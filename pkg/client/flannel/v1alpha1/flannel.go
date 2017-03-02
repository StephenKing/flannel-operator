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
	TPRFlannelKind = "Flanel"
	TPRFlannelName = "flannels"
)

type FlannelsGetter interface {
	Flannels(namespace string) FlannelInterface
}

type FlannelInterface interface {
	Create(*Flannel) (*Flannel, error)
	Get(name string) (*Flannel, error)
	Update(*Flannel) (*Flannel, error)
	Delete(name string, options *v1.DeleteOptions)
	List(opts api.ListOptions) (runtime.Object, error)
	Watch(opts api.ListOptions) (watch.Interface, error)
}

type flannels struct {
	restClient *rest.RESTClient
	client     *dynamic.ResourceClient
	ns         string
}

func newFlannels(r *rest.RESTClient, c *dynamic.Client, namespace string) *flannels {
	return &flannels{
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

func (f *flannels) Create(o *Flannel) (*Flannel, error) {
	up, err := UnstructuredFromFlannel(o)
	if err != nil {
		return nil, err
	}

	up, err = f.client.Create(up)
	if err != nil {
		return nil, err
	}

	return FlannelFromUnstructured(up)
}

func (f *flannels) Get(name string) (*Flannel, error) {
	obj, err := f.client.Get(name)
	if err != nil {
		return nil, err
	}
	return FlannelFromUnstructured(obj)
}

func (f *flannels) Update(o *Flannel) (*Flannel, error) {
	up, err := UnstructuredFromFlannel(o)
	if err != nil {
		return nil, err
	}

	up, err = f.client.Update(up)
	if err != nil {
		return nil, err
	}

	return FlannelFromUnstructured(up)
}

// TODO had to remove the return type "error" because of
// pkg/client/flannel/v1alpha1/client.go:25: cannot use newFlannels(c.restClient, c.dynamicClient, namespace) (type *flannels) as type FlannelInterface in return argument:
func (f *flannels) Delete(name string, options *v1.DeleteOptions) {
	if err := f.client.Delete(name, options); err != nil {
		log.Error("Could not delete %v - %v", name, err)
	}

}

func (f *flannels) List(opts api.ListOptions) (runtime.Object, error) {
	req := f.restClient.Get().
		Namespace(f.ns).
		Resource("flannels").
		FieldsSelectorParam(nil)

	b, err := req.DoRaw()
	if err != nil {
		return nil, err
	}
	var flan FlannelList
	return &flan, json.Unmarshal(b, &flan)
}

func (f *flannels) Watch(opts api.ListOptions) (watch.Interface, error) {
	r, err := f.restClient.Get().
		Prefix("watch").
		Namespace(f.ns).
		Resource("flannels").
	// VersionedParams(&options, v1.ParameterCodec).
		FieldsSelectorParam(nil).
		Stream()
	if err != nil {
		return nil, err
	}
	return watch.NewStreamWatcher(&flannelDecoder{
		dec:   json.NewDecoder(r),
		close: r.Close,
	}), nil
}

// FlannelFromUnstructured unmarshals a Flannel object from dynamic client's unstructured
func FlannelFromUnstructured(r *runtime.Unstructured) (*Flannel, error) {
	b, err := json.Marshal(r.Object)
	if err != nil {
		return nil, err
	}
	var f Flannel
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, err
	}
	f.TypeMeta.Kind = TPRFlannelKind
	f.TypeMeta.APIVersion = TPRGroup + "/" + TPRVersion
	return &f, nil
}

// UnstructuredFromFlannel marshals a Flannel object into dynamic client's unstructured
func UnstructuredFromFlannel(f *Flannel) (*runtime.Unstructured, error) {
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

type flannelDecoder struct {
	dec   *json.Decoder
	close func() error
}

func (d *flannelDecoder) Close() {
	d.close()
}

func (d *flannelDecoder) Decode() (action watch.EventType, object runtime.Object, err error) {
	var e struct {
		Type   watch.EventType
		Object Flannel
	}
	if err := d.dec.Decode(&e); err != nil {
		return watch.Error, nil, err
	}
	return e.Type, &e.Object, nil
}
