package k8s

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	kubeyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ParseError struct {
	Data []byte
	Err  error
}

func (p *ParseError) Error() string {
	return fmt.Sprintf("error parsing data %s: %v", string(p.Data), p.Err.Error())
}

func commentOnly(d []byte) bool {
	for _, b := range bytes.Split(d, []byte("\n")) {
		line := strings.TrimPrefix(string(b), " ")
		if line != "" && !strings.HasPrefix(line, "#") {
			return false
		}
	}
	return true
}

func ParseUnstructured(r io.Reader) ([]*unstructured.Unstructured, error) {
	objects, err := ParseKubernetesObjects(r, &unstructured.Unstructured{})
	if err != nil {
		return nil, err
	}
	ret := []*unstructured.Unstructured{}
	for _, o := range objects {
		u, ok := o.(*unstructured.Unstructured)
		if !ok {
			return nil, fmt.Errorf("unexpected type %T expecting *unstructured.Unstructured", o)
		}
		ret = append(ret, u)
	}
	return ret, nil
}

func ParseKubernetesObjects(r io.Reader, as runtime.Object) ([]runtime.Object, error) {
	objects := []runtime.Object{}
	kubereader := kubeyaml.NewYAMLReader(bufio.NewReader(r))
	for {
		data, err := kubereader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return []runtime.Object{}, err
		}
		data = bytes.TrimLeft(data, "---")
		if !commentOnly(data) {
			if as != nil {
				as = as.DeepCopyObject()
			}
			o, _, err := scheme.Codecs.UniversalDeserializer().Decode(data, nil, as)
			if err != nil {
				return []runtime.Object{}, &ParseError{
					Data: data,
					Err:  err,
				}
			}
			objects = append(objects, o)
		}
	}
	return objects, nil
}

func SerialiseObjects(scheme *runtime.Scheme, w io.Writer, objects ...runtime.Object) error {
	for i, o := range objects {
		if i > 0 {
			w.Write([]byte("---\n"))
		}
		err := serializer.NewCodecFactory(scheme).WithoutConversion().EncoderForVersion(
			json.NewSerializerWithOptions(
				json.DefaultMetaFactory,
				scheme,
				scheme,
				json.SerializerOptions{
					Yaml:   true,
					Strict: true,
					Pretty: true,
				}),
			nil,
		).Encode(o, w)
		if err != nil {
			return err
		}
	}
	return nil
}

func ToUnstructured(scheme *runtime.Scheme, objects ...client.Object) ([]*unstructured.Unstructured, error) {
	unstructuredObjects := []*unstructured.Unstructured{}
	for _, obj := range objects {
		switch o := obj.(type) {
		case *unstructured.Unstructured:
			unstructuredObjects = append(unstructuredObjects, o)
		default:
			data, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
			if err != nil {
				return nil, err
			}
			gvks, _, err := scheme.ObjectKinds(obj)
			if err != nil {
				return nil, err
			}
			if len(gvks) == 0 {
				return nil, fmt.Errorf("Unable to find group version kind for obkect %T", obj)
			}
			u := &unstructured.Unstructured{Object: data}
			u.GetObjectKind().SetGroupVersionKind(gvks[0])
			if err != nil {
				return nil, err
			}
			unstructuredObjects = append(unstructuredObjects, u)
		}
	}
	return unstructuredObjects, nil
}

func ToClientObject(unstructuredObjects []*unstructured.Unstructured) []client.Object {
	r := []client.Object{}
	for _, o := range unstructuredObjects {
		r = append(r, o)
	}
	return r
}
