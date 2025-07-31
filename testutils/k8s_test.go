package testutils_test

import (
	"context"
	"errors"
	"testing"
	"time"

	k8stestutils "github.com/adevinta/go-k8s-toolkit/testutils"
	testutils "github.com/adevinta/go-testutils-toolkit"
	"github.com/cenkalti/backoff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func TestCreateOrUpdateAll(t *testing.T) {
	t.Run("when objects do not exist", func(t *testing.T) {
		ctx := context.Background()
		ft := &testutils.FakeTest{}
		ns := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-ns",
			},
		}
		cm := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "my-ns",
				Name:      "my-cm",
			},
		}
		c := fake.NewClientBuilder().Build()
		k8stestutils.CreateOrUpdateAll(ft, ctx, c, ns, cm)
		assert.Len(t, ft.ErrorMessages, 0)
		assert.False(t, ft.Failed)
		k8stestutils.AssertHasObject(t, c, ns)
		k8stestutils.AssertHasObject(t, c, cm)
	})

	t.Run("when objects already exists", func(t *testing.T) {
		ctx := context.Background()
		ft := &testutils.FakeTest{}
		ns := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-ns",
			},
		}
		cm := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "my-ns",
				Name:      "my-cm",
			},
			Data: map[string]string{},
		}
		c := fake.NewClientBuilder().WithObjects(ns.DeepCopy(), cm.DeepCopy()).Build()
		updatedCM := cm.DeepCopy()
		updatedCM.Data["my-data"] = "new version"
		k8stestutils.CreateOrUpdateAll(ft, ctx, c, ns, updatedCM)
		assert.Len(t, ft.ErrorMessages, 0)
		assert.False(t, ft.Failed)
		k8stestutils.AssertHasObject(t, c, ns)
		k8stestutils.AssertHasObject(t, c, cm)

		assert.Equal(t, updatedCM.Data, cm.Data)
	})

	t.Run("when cm creation fails the first time", func(t *testing.T) {
		t.Cleanup(k8stestutils.ResetHooks)
		k8stestutils.SetNewBackOff(func() backoff.BackOff {
			return backoff.WithMaxRetries(&backoff.ConstantBackOff{
				Interval: 10 * time.Millisecond,
			}, 5)
		})
		ctx := context.Background()
		ft := &testutils.FakeTest{}
		cm := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "my-ns",
				Name:      "my-cm",
			},
			Data: map[string]string{},
		}
		ns := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-ns",
			},
		}
		k8sClient := fake.NewClientBuilder().Build()
		c := &ClientUpdateFuncs{
			Client: k8sClient,
			createFunc: func(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
				if obj.GetNamespace() != "" {
					ns := &v1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Name: obj.GetNamespace(),
						},
					}
					err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ns), ns)
					if apierrors.IsNotFound(err) {
						return apierrors.NewConflict(schema.GroupResource{}, obj.GetName(), errors.New("namespace missing"))
					}
				}
				return k8sClient.Create(ctx, obj, opts...)
			},
		}
		k8stestutils.CreateOrUpdateAll(ft, ctx, c, cm, ns)
		assert.False(t, ft.Failed)

		assert.Len(t, ft.ErrorMessages, 0)
		assert.Equal(t, 4, c.getCalls)
		assert.Equal(t, 3, c.createCalls) // 2 success, 1 failure
		assert.Equal(t, 1, c.updateCalls)
	})

	t.Run("when cm creation fails permanently", func(t *testing.T) {
		t.Cleanup(k8stestutils.ResetHooks)
		k8stestutils.SetNewBackOff(func() backoff.BackOff {
			return backoff.WithMaxRetries(&backoff.ConstantBackOff{
				Interval: 10 * time.Millisecond,
			}, 5)
		})
		ctx := context.Background()
		ft := &testutils.FakeTest{}
		cm := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "my-ns",
				Name:      "my-cm",
			},
			Data: map[string]string{},
		}
		ns := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-ns",
				// fake client sets creationTimeStamp to nil
				// the unstructured library seems to avoid setting the creationTimeStamp to nil
				CreationTimestamp: metav1.Now(),
			},
		}
		k8sClient := fake.NewClientBuilder().Build()
		c := &ClientUpdateFuncs{
			Client: k8sClient,
			getFunc: func(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				if obj.GetObjectKind().GroupVersionKind().Kind != "ConfigMap" {
					return k8sClient.Get(ctx, key, obj)
				}
				return apierrors.NewNotFound(schema.GroupResource{}, obj.GetName())
			},
			createFunc: func(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
				if obj.GetObjectKind().GroupVersionKind().Kind != "ConfigMap" {
					return k8sClient.Create(ctx, obj, opts...)
				}
				return errors.New("test-error")
			},
		}
		k8stestutils.CreateOrUpdateAll(ft, ctx, c, cm, ns)
		assert.True(t, ft.Failed)

		require.Len(t, ft.ErrorMessages, 1)
		assert.Contains(t, ft.ErrorMessages[0], "new retry did not manage to create any new dependent objects. Failing hard")
	})
}

func TestExcludeObjectWithKind(t *testing.T) {
	t.Run("When the kind matches", func(t *testing.T) {
		o := &unstructured.Unstructured{Object: map[string]interface{}{"kind": "Namespace"}}
		f := k8stestutils.ExcludeObject(k8stestutils.WithKind("Namespace"))
		assert.False(t, f.Include(o))
		assert.EqualValues(t, o, f.Mutate(o))
	})
	t.Run("When the kind differs", func(t *testing.T) {
		o := &unstructured.Unstructured{Object: map[string]interface{}{"kind": "ConfigMap"}}
		f := k8stestutils.ExcludeObject(k8stestutils.WithKind("Namespace"))
		assert.True(t, f.Include(o))
		assert.EqualValues(t, o, f.Mutate(o))
	})
}

func TestAndObjectSelector(t *testing.T) {
	assert.True(t, k8stestutils.And()(&unstructured.Unstructured{}))
	assert.True(t, k8stestutils.And(
		func(object *unstructured.Unstructured) bool { return true },
		func(object *unstructured.Unstructured) bool { return true },
		func(object *unstructured.Unstructured) bool { return true },
		func(object *unstructured.Unstructured) bool { return true },
	)(&unstructured.Unstructured{}),
	)
	assert.False(t, k8stestutils.And(
		func(object *unstructured.Unstructured) bool { return true },
		func(object *unstructured.Unstructured) bool { return true },
		func(object *unstructured.Unstructured) bool { return false },
		func(object *unstructured.Unstructured) bool { return true },
	)(&unstructured.Unstructured{}),
	)
	assert.False(t, k8stestutils.And(
		func(object *unstructured.Unstructured) bool { return true },
		func(object *unstructured.Unstructured) bool { return true },
		func(object *unstructured.Unstructured) bool { return false },
	)(&unstructured.Unstructured{}),
	)
}

func TestWithKind(t *testing.T) {
	assert.False(t, k8stestutils.WithKind("ConfigMap")(&unstructured.Unstructured{Object: map[string]interface{}{"kind": "Namespace"}}))
	assert.True(t, k8stestutils.WithKind("ConfigMap")(&unstructured.Unstructured{Object: map[string]interface{}{"kind": "ConfigMap"}}))
}

func TestWithName(t *testing.T) {
	assert.False(t, k8stestutils.WithName("my-name")(&unstructured.Unstructured{Object: map[string]interface{}{"metadata": map[string]interface{}{"name": "other-name"}}}))
	assert.True(t, k8stestutils.WithName("my-name")(&unstructured.Unstructured{Object: map[string]interface{}{"metadata": map[string]interface{}{"name": "my-name"}}}))
}

func TestWithNamespace(t *testing.T) {
	assert.False(t, k8stestutils.WithNamespace("my-namespace")(&unstructured.Unstructured{Object: map[string]interface{}{"metadata": map[string]interface{}{"namespace": "other-namespace"}}}))
	assert.True(t, k8stestutils.WithNamespace("my-namespace")(&unstructured.Unstructured{Object: map[string]interface{}{"metadata": map[string]interface{}{"namespace": "my-namespace"}}}))
}

func TestExtractObjectName(t *testing.T) {
	t.Run("When the object matches", func(t *testing.T) {
		name := "not-found"
		o := &unstructured.Unstructured{Object: map[string]interface{}{"kind": "Namespace", "metadata": map[string]interface{}{"name": "my-name"}}}
		f := k8stestutils.ExtractObjectName(k8stestutils.WithKind("Namespace"), &name)
		f.Mutate(o)
		assert.True(t, f.Include(o))
		assert.Equal(t, "my-name", name)
	})
	t.Run("When the object does not match", func(t *testing.T) {
		name := "not-found"
		f := k8stestutils.ExtractObjectName(k8stestutils.WithKind("ConfigMap"), &name)
		o := &unstructured.Unstructured{Object: map[string]interface{}{"kind": "Namespace", "metadata": map[string]interface{}{"name": "my-name"}}}
		f.Mutate(o)
		assert.True(t, f.Include(o))
		assert.Equal(t, "not-found", name)
	})
}

func TestExtractObjectNamespace(t *testing.T) {
	t.Run("When the object matches", func(t *testing.T) {
		namespace := "not-found"
		o := &unstructured.Unstructured{Object: map[string]interface{}{"kind": "ConfigMap", "metadata": map[string]interface{}{"name": "my-name", "namespace": "my-namespace"}}}
		f := k8stestutils.ExtractObjectNamespace(k8stestutils.WithKind("ConfigMap"), &namespace)
		f.Mutate(o)
		assert.True(t, f.Include(o))
		assert.Equal(t, "my-namespace", namespace)
	})
	t.Run("When the object does not match", func(t *testing.T) {
		namespace := "not-found"
		f := k8stestutils.ExtractObjectNamespace(k8stestutils.WithKind("Secret"), &namespace)
		o := &unstructured.Unstructured{Object: map[string]interface{}{"kind": "ConfigMap", "metadata": map[string]interface{}{"name": "my-name", "namespace": "my-namespace"}}}
		f.Mutate(o)
		assert.True(t, f.Include(o))
		assert.Equal(t, "not-found", namespace)
	})
}

func TestExtractObjectField(t *testing.T) {
	t.Run("When the field does not exist", func(t *testing.T) {
		ft := &testutils.FakeTest{}
		o := &unstructured.Unstructured{Object: map[string]interface{}{"kind": "ConfigMap", "metadata": map[string]interface{}{"name": "my-name", "namespace": "my-namespace"}}}
		data := "not-found"
		f := k8stestutils.ExtractObjectField(ft, k8stestutils.WithKind("ConfigMap"), &data, "path", "does", "not", "exist")
		f.Mutate(o)
		assert.True(t, f.Include(o))
		assert.Equal(t, "not-found", data)
		assert.True(t, ft.Failed)
		require.Len(t, ft.ErrorMessages, 1)
		assert.Contains(t, ft.ErrorMessages[0], "path.does.not.exist")
	})
	t.Run("When the field has invalid type", func(t *testing.T) {
		ft := &testutils.FakeTest{}
		o := &unstructured.Unstructured{Object: map[string]interface{}{
			"kind": "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "my-name",
				"namespace": "my-namespace",
				"labels": map[string]string{
					"a-label": "value 1",
					"other":   "world",
				},
			},
		},
		}
		data := "not-found"
		f := k8stestutils.ExtractObjectField(ft, k8stestutils.WithKind("ConfigMap"), &data, "metadata", "labels")
		f.Mutate(o)
		assert.True(t, f.Include(o))
		assert.Equal(t, "not-found", data)
		assert.True(t, ft.Failed)
		require.Len(t, ft.ErrorMessages, 1)
		assert.Contains(t, ft.ErrorMessages[0], "Unexpected type map[string]string for field metadata.labels")
	})
	t.Run("When the field has invalid type", func(t *testing.T) {
		ft := &testutils.FakeTest{}
		o := &unstructured.Unstructured{Object: map[string]interface{}{
			"kind": "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "my-name",
				"namespace": "my-namespace",
				"labels": map[string]string{
					"a-label": "value 1",
					"other":   "world",
				},
			},
		},
		}
		data := map[string]string{}
		f := k8stestutils.ExtractObjectField(ft, k8stestutils.WithKind("ConfigMap"), &data, "metadata", "labels")
		f.Mutate(o)
		assert.True(t, f.Include(o))
		assert.False(t, ft.Failed)
		assert.Equal(
			t,
			map[string]string{
				"a-label": "value 1",
				"other":   "world",
			},
			data,
		)
	})
}

func TestFilterUnstructuredObjects(t *testing.T) {
	assert.Equal(
		t,
		[]*unstructured.Unstructured{
			{
				Object: map[string]interface{}{
					"kind": "Namespace",
					"metadata": map[string]interface{}{
						"name": "namespace",
					},
				},
			},
			{
				Object: map[string]interface{}{
					"kind": "ConfigMap",
					"metadata": map[string]interface{}{
						"name": "mutated",
					},
					"data": map[string]string{
						"some": "data",
					},
				},
			},
		},
		k8stestutils.FilterUnstructuredObjects(
			[]*unstructured.Unstructured{
				{
					Object: map[string]interface{}{
						"kind": "Namespace",
						"metadata": map[string]interface{}{
							"name": "namespace",
						},
					},
				},
				{
					Object: map[string]interface{}{
						"kind": "ConfigMap",
						"metadata": map[string]interface{}{
							"name": "other",
						},
					},
				},
				{
					Object: map[string]interface{}{
						"kind": "ConfigMap",
						"metadata": map[string]interface{}{
							"name": "my-cm",
						},
					},
				},
			},
			k8stestutils.ExcludeObject(k8stestutils.And(
				k8stestutils.WithKind("ConfigMap"),
				k8stestutils.WithName("my-cm"),
			)),
			k8stestutils.MutateFilterFunc(func(u *unstructured.Unstructured) *unstructured.Unstructured {
				if u.GetKind() == "ConfigMap" {
					return &unstructured.Unstructured{
						Object: map[string]interface{}{
							"kind": "ConfigMap",
							"metadata": map[string]interface{}{
								"name": "mutated",
							},
							"data": map[string]string{
								"some": "data",
							},
						},
					}
				}
				return u
			}),
		),
	)
}

type ClientUpdateFuncs struct {
	client.Client
	getFunc     func(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error
	createFunc  func(ctx context.Context, obj client.Object, opts ...client.CreateOption) error
	updateFunc  func(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error
	getCalls    int
	createCalls int
	updateCalls int
}

var _ client.Client = &ClientUpdateFuncs{}

func (c *ClientUpdateFuncs) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	c.getCalls++
	if c.getFunc != nil {
		err := c.getFunc(ctx, key, obj, opts...)
		if err != nil {
			return err
		}
		return nil
	}
	if c.Client != nil {
		err := c.Client.Get(ctx, key, obj, opts...)
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("Get not available, createFunc and Client are nil")
}

func (c *ClientUpdateFuncs) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	c.createCalls++
	if c.createFunc != nil {
		err := c.createFunc(ctx, obj, opts...)
		if err != nil {
			return err
		}
		return nil
	}
	if c.Client != nil {

		err := c.Client.Create(ctx, obj, opts...)
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("Create not available, createFunc and Client are nil")
}
func (c *ClientUpdateFuncs) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	c.updateCalls++
	if c.updateFunc != nil {
		err := c.updateFunc(ctx, obj, opts...)
		if err != nil {
			return err
		}
		return nil
	}
	if c.Client != nil {
		err := c.Client.Update(ctx, obj, opts...)
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("Update not available, createFunc and Client are nil")
}
