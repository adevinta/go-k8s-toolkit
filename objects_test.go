package k8s_test

import (
	"bytes"
	"strings"
	"testing"

	k8s "github.com/adevinta/go-k8s-toolkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const testObjects = `
---

---
# some comment
---
# some/file.yaml
---
apiVersion: v1
kind: Namespace
metadata:
  name: some-name
  labels:
    name: some-name
---
---
apiVersion: v1
kind: Pod
metadata:
  name: pod-name
  namespace: pod-namespace
  labels:
    name: pod-name
`

func TestParseUnstructured(t *testing.T) {
	o, err := k8s.ParseUnstructured(strings.NewReader(testObjects))
	require.NoError(t, err)
	require.NotNil(t, o)
	require.Len(t, o, 2)
	assert.Equal(t, schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Namespace"}, o[0].GetObjectKind().GroupVersionKind())
	assert.Equal(t, schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"}, o[1].GetObjectKind().GroupVersionKind())
}

func TestSerializeObjects(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	d := bytes.Buffer{}
	assert.NoError(t, k8s.SerialiseObjects(
		scheme,
		&d,
		&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-namespace",
			},
		},
		&v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-cm",
				Namespace: "my-namespace",
			},
			Data: map[string]string{
				"hello": "world",
			},
		},
		&unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Secret",
				"metadata": map[string]interface{}{
					"name":      "my-secret",
					"namespace": "my-namespace",
				},
			},
		},
	))
	o, err := k8s.ParseUnstructured(&d)
	require.NoError(t, err)
	assert.EqualValues(t,
		[]*unstructured.Unstructured{
			{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Namespace",
					"metadata": map[string]interface{}{
						"name":              "my-namespace",
						"creationTimestamp": nil,
					},
					"spec":   map[string]interface{}{},
					"status": map[string]interface{}{},
				},
			},
			{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "ConfigMap",
					"metadata": map[string]interface{}{
						"name":              "my-cm",
						"namespace":         "my-namespace",
						"creationTimestamp": nil,
					},
					"data": map[string]interface{}{
						"hello": "world",
					},
				},
			},
			{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Secret",
					"metadata": map[string]interface{}{
						"name":      "my-secret",
						"namespace": "my-namespace",
					},
				},
			},
		},
		o,
	)
}

func TestParseObjectsHandlesUnstructuredObjects(t *testing.T) {
	o, err := k8s.ParseKubernetesObjects(strings.NewReader(testObjects), &unstructured.Unstructured{})
	require.NoError(t, err)
	require.NotNil(t, o)
	require.Len(t, o, 2)
	assert.IsType(t, &unstructured.Unstructured{}, o[0])
	assert.IsType(t, &unstructured.Unstructured{}, o[1])
	assert.Equal(t, schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Namespace"}, o[0].GetObjectKind().GroupVersionKind())
	assert.Equal(t, schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"}, o[1].GetObjectKind().GroupVersionKind())
}

func TestParseObjectsReturnsStructuredObjects(t *testing.T) {
	o, err := k8s.ParseKubernetesObjects(strings.NewReader(testObjects), nil)
	require.NoError(t, err)
	require.NotNil(t, o)
	assert.Equal(t, []runtime.Object{
		&v1.Namespace{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Namespace",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "some-name",
				Labels: map[string]string{
					"name": "some-name",
				},
			},
		},
		&v1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod-name",
				Namespace: "pod-namespace",
				Labels: map[string]string{
					"name": "pod-name",
				},
			},
		},
	}, o)
}

func TestToClient(t *testing.T) {
	assert.EqualValues(t,
		[]client.Object{
			&unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "custom.testing.ltd/v1",
					"kind":       "Custom",
					"metadata": map[string]interface{}{
						"namespace": "my-namespace",
						"name":      "my-custom",
					},
				},
			},
		},
		k8s.ToClientObject([]*unstructured.Unstructured{
			{
				Object: map[string]interface{}{
					"apiVersion": "custom.testing.ltd/v1",
					"kind":       "Custom",
					"metadata": map[string]interface{}{
						"namespace": "my-namespace",
						"name":      "my-custom",
					},
				},
			},
		},
		),
	)
}

func TestToUnstructured(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	objects, err := k8s.ToUnstructured(scheme,
		&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-namespace",
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "my-namespace",
				Name:      "my-deployment",
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"test": "app",
					},
				},
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name: "test",
							},
						},
					},
				},
			},
		},
		&unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "custom.testing.ltd/v1",
				"kind":       "Custom",
				"metadata": map[string]interface{}{
					"namespace": "my-namespace",
					"name":      "my-custom",
				},
			},
		},
	)
	require.NoError(t, err)
	assert.EqualValues(
		t,
		[]*unstructured.Unstructured{
			{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Namespace",
					"metadata": map[string]interface{}{
						"creationTimestamp": nil,
						"name":              "my-namespace",
					},
					"spec":   map[string]interface{}{},
					"status": map[string]interface{}{},
				},
			},
			{
				Object: map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "Deployment",
					"metadata": map[string]interface{}{
						"creationTimestamp": nil,
						"namespace":         "my-namespace",
						"name":              "my-deployment",
					},
					"spec": map[string]interface{}{
						"selector": map[string]interface{}{
							"matchLabels": map[string]interface{}{
								"test": "app",
							},
						},
						"strategy": map[string]interface{}{},
						"template": map[string]interface{}{
							"metadata": map[string]interface{}{
								"creationTimestamp": nil,
							},
							"spec": map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"name":      "test",
										"resources": map[string]interface{}{},
									},
								},
							},
						},
					},
					"status": map[string]interface{}{},
				},
			},
			{
				Object: map[string]interface{}{
					"apiVersion": "custom.testing.ltd/v1",
					"kind":       "Custom",
					"metadata": map[string]interface{}{
						"namespace": "my-namespace",
						"name":      "my-custom",
					},
				},
			},
		},
		objects,
	)
}
