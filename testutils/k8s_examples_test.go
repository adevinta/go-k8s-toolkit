package testutils_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	k8stestutils "github.com/adevinta/go-k8s-toolkit/testutils"
	testutils "github.com/adevinta/go-testutils-toolkit"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func ExampleCreateOrUpdateAll() {
	// use the real *testing.T from the test
	t := &testutils.FakeTest{Name: "TestICanCreateObjects"}

	client := fake.NewClientBuilder().Build()

	k8stestutils.CreateOrUpdateAll(t, context.Background(), client, &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-namespace",
		},
	})

	k8stestutils.AssertHasObject(t, client, &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-namespace",
		},
	})

	// Usually, this is done by the go framework
	fmt.Println(t)
	// Output: --- PASS: TestICanCreateObjects
}

func ExampleExtractObjectName() {
	name := "unknown"

	myObjects := []*unstructured.Unstructured{
		{
			Object: map[string]interface{}{
				"kind": "ConfigMap",
				"metadata": map[string]interface{}{
					"namespace": "my-namespace",
					"name":      "my-cm",
				},
			},
		},
		{
			Object: map[string]interface{}{
				"kind": "Secret",
				"metadata": map[string]interface{}{
					"namespace": "my-namespace",
					"name":      "my-secret",
				},
			},
		},
	}

	k8stestutils.FilterUnstructuredObjects(myObjects, k8stestutils.ExtractObjectName(k8stestutils.WithKind("ConfigMap"), &name))

	fmt.Println(name)
	// Output: my-cm
}

func ExampleExtractObjectNamespace() {
	namespace := "unknown"

	myObjects := []*unstructured.Unstructured{
		{
			Object: map[string]interface{}{
				"kind": "ConfigMap",
				"metadata": map[string]interface{}{
					"namespace": "my-namespace",
					"name":      "my-cm",
				},
			},
		},
		{
			Object: map[string]interface{}{
				"kind": "ConfigMap",
				"metadata": map[string]interface{}{
					"namespace": "other",
					"name":      "other",
				},
			},
		},
	}

	k8stestutils.FilterUnstructuredObjects(myObjects, k8stestutils.ExtractObjectNamespace(k8stestutils.And(k8stestutils.WithKind("ConfigMap"), k8stestutils.WithName("my-cm")), &namespace))

	fmt.Println(namespace)
	// Output: my-namespace
}

func ExampleExtractObjectField() {
	// use the real *testing.T from the test
	t := &testutils.FakeTest{Name: "TestICanExtractObjectFields"}
	labels := map[string]string{}

	myObjects := []*unstructured.Unstructured{
		{
			Object: map[string]interface{}{
				"kind": "ConfigMap",
				"metadata": map[string]interface{}{
					"namespace": "my-namespace",
					"name":      "my-cm",
					"labels": map[string]string{
						"my": "label",
					},
				},
			},
		},
		{
			Object: map[string]interface{}{
				"kind": "ConfigMap",
				"metadata": map[string]interface{}{
					"namespace": "other",
					"name":      "other",
				},
			},
		},
	}

	k8stestutils.FilterUnstructuredObjects(myObjects, k8stestutils.ExtractObjectField(t, k8stestutils.And(k8stestutils.WithKind("ConfigMap"), k8stestutils.WithName("my-cm")), &labels, "metadata", "labels"))

	json.NewEncoder(os.Stdout).Encode(labels)
	// Output: {"my":"label"}
}

// start ReadMe examples

func TestICanCreateObjects(t *testing.T) {
	client := fake.NewClientBuilder().Build()

	k8stestutils.CreateOrUpdateAll(t, context.Background(), client, &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-namespace",
		},
	})

	k8stestutils.AssertHasObject(t, client, &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-namespace",
		},
	})
}

func TestICanExtractObjectFields(t *testing.T) {
	labels := map[string]string{}

	myObjects := []*unstructured.Unstructured{
		{
			Object: map[string]interface{}{
				"kind": "ConfigMap",
				"metadata": map[string]interface{}{
					"namespace": "my-namespace",
					"name":      "my-cm",
					"labels": map[string]string{
						"my": "label",
					},
				},
			},
		},
		{
			Object: map[string]interface{}{
				"kind": "ConfigMap",
				"metadata": map[string]interface{}{
					"namespace": "other",
					"name":      "other",
				},
			},
		},
	}

	k8stestutils.FilterUnstructuredObjects(myObjects, k8stestutils.ExtractObjectField(t, k8stestutils.And(k8stestutils.WithKind("ConfigMap"), k8stestutils.WithName("my-cm")), &labels, "metadata", "labels"))

	assert.Equal(t, map[string]string{"my": "label"}, labels)
}
