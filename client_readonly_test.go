package k8s_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	k8s "github.com/adevinta/go-k8s-toolkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	ctrl "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func testAllWriteMethodsAreImplemented(t *testing.T, client interface{}, knownReadOnlyMethods map[string]struct{}) {
	t.Helper()
	t.Run(fmt.Sprintf("%T", client), func(t *testing.T) {
		require.NotNil(t, client)

		tpe := reflect.TypeOf(client)
		require.NotNil(t, tpe)

		for i := 0; i < tpe.NumMethod(); i++ {
			method := tpe.Method(i)
			t.Run(method.Name, func(t *testing.T) {
				args := []reflect.Value{}
				for i := 0; i < method.Type.NumIn(); i++ {
					argType := method.Type.In(i)
					if !method.Type.IsVariadic() || i < method.Type.NumIn()-1 {
						args = append(args, reflect.New(argType).Elem())
					}
				}
				if _, ok := knownReadOnlyMethods[method.Name]; ok {
					// Panics as the original client is a nil pointer
					assert.Panicsf(t, func() {
						method.Func.Call(args)
					},
						"The original client should be called for known read-only method %s",
						method.Name,
					)
				} else {
					assert.NotPanicsf(t, func() {
						method.Func.Call(args)
					},
						"The original client should not be called for write method %s",
						method.Name,
					)
				}
			})
		}
	})
}

func TestAllWriteFieldsAreImplemented(t *testing.T) {

	client := k8s.ReadOnlyClient(nil)
	client.Status()
	testAllWriteMethodsAreImplemented(t, client, map[string]struct{}{
		"Get":                 {},
		"List":                {},
		"RESTMapper":          {},
		"Scheme":              {},
		"Status":              {}, // This one is tested right below
		"SubResource":         {}, // This one is tested right below
		"GroupVersionKindFor": {},
		"IsObjectNamespaced":  {},
	})
	testAllWriteMethodsAreImplemented(t, client.Status(), map[string]struct{}{
		"Get":  {},
		"List": {},
	})
	testAllWriteMethodsAreImplemented(t, client.SubResource("any"), map[string]struct{}{
		"Get":  {},
		"List": {},
	})
}

func TestReadOnlyClientDoesNotCallCreate(t *testing.T) {
	client := k8s.ReadOnlyClient(fake.NewClientBuilder().Build())
	err := client.Create(context.Background(), &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test"}})
	assert.Error(t, err)
	assert.Equal(t, "Create not allowed in read-only mode", err.Error())
	namespaces := &v1.NamespaceList{}
	require.NoError(t, client.List(context.Background(), namespaces))
	assert.Empty(t, namespaces.Items)
}

func TestReadOnlyClientDoesNotCallUpdate(t *testing.T) {
	ns := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test"}}

	client := k8s.ReadOnlyClient(fake.NewClientBuilder().WithObjects(ns).Build())
	_, err := ctrl.CreateOrUpdate(context.Background(), client, ns, func() error {
		ns.Labels = map[string]string{"test": "test"}
		return nil
	})
	assert.Error(t, err)
	assert.Equal(t, "Update not allowed in read-only mode", err.Error())
	namespaces := &v1.NamespaceList{}
	require.NoError(t, client.List(context.Background(), namespaces))
	require.Len(t, namespaces.Items, 1)
	assert.Equal(t, "test", namespaces.Items[0].Name)
	assert.Empty(t, namespaces.Items[0].Labels)
}

func TestReadOnlyClientDoesNotCallPatch(t *testing.T) {
	ns := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test"}}

	client := k8s.ReadOnlyClient(fake.NewClientBuilder().WithObjects(ns).Build())
	_, err := ctrl.CreateOrPatch(context.Background(), client, ns, func() error {
		ns.Labels = map[string]string{"test": "test"}
		return nil
	})
	assert.Error(t, err)
	assert.Equal(t, "Patch not allowed in read-only mode", err.Error())
	namespaces := &v1.NamespaceList{}
	require.NoError(t, client.List(context.Background(), namespaces))
	require.Len(t, namespaces.Items, 1)
	assert.Equal(t, "test", namespaces.Items[0].Name)
	assert.Empty(t, namespaces.Items[0].Labels)
}

func TestReadOnlyClientDoesNotCallDelete(t *testing.T) {
	ns := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test"}}

	client := k8s.ReadOnlyClient(fake.NewClientBuilder().WithObjects(ns).Build())
	err := client.Delete(context.Background(), ns)
	assert.Error(t, err)
	assert.Equal(t, "Delete not allowed in read-only mode", err.Error())
	namespaces := &v1.NamespaceList{}
	require.NoError(t, client.List(context.Background(), namespaces))
	require.Len(t, namespaces.Items, 1)
	assert.Equal(t, "test", namespaces.Items[0].Name)
}

func TestReadOnlyClientDoesNotCallDeleteAllOf(t *testing.T) {
	ns := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test"}}

	client := k8s.ReadOnlyClient(fake.NewClientBuilder().WithObjects(&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test"}}, &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-2"}}).Build())
	err := client.DeleteAllOf(context.Background(), ns)
	assert.Error(t, err)
	assert.Equal(t, "DeleteAllOf not allowed in read-only mode", err.Error())
	namespaces := &v1.NamespaceList{}
	require.NoError(t, client.List(context.Background(), namespaces))
	require.Len(t, namespaces.Items, 2)
}

func TestReadOnlySubResourceClientDoesNotCallCreate(t *testing.T) {
	pod := &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"}}
	cl := k8s.ReadOnlyClient(fake.NewClientBuilder().WithObjects(pod).Build())

	assert.Error(t, cl.Status().Create(context.Background(), pod, &v1.Pod{Status: v1.PodStatus{Phase: v1.PodRunning}}))

	assert.NoError(t, cl.Get(context.Background(), client.ObjectKeyFromObject(pod), pod))
	assert.Empty(t, pod.Status.Phase)

}

func TestReadOnlySubResourceClientDoesNotCallUpdate(t *testing.T) {
	pod := &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"}}
	cl := k8s.ReadOnlyClient(fake.NewClientBuilder().WithObjects(pod).Build())

	assert.Error(t, cl.Status().Update(context.Background(), &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"}, Status: v1.PodStatus{Phase: v1.PodRunning}}))

	assert.NoError(t, cl.Get(context.Background(), client.ObjectKeyFromObject(pod), pod))
	assert.Empty(t, pod.Status.Phase)
}

func TestReadOnlySubResourceClientDoesNotCallPatch(t *testing.T) {
	pod := &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"}}
	cl := k8s.ReadOnlyClient(fake.NewClientBuilder().WithObjects(pod).Build())

	assert.Error(t, cl.Status().Patch(context.Background(), &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"}, Status: v1.PodStatus{Phase: v1.PodRunning}}, client.MergeFrom(pod)))

	assert.NoError(t, cl.Get(context.Background(), client.ObjectKeyFromObject(pod), pod))
	assert.Empty(t, pod.Status.Phase)
}
