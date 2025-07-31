package testutils_test

import (
	"testing"

	k8stestutils "github.com/adevinta/go-k8s-toolkit/testutils"
	testutils "github.com/adevinta/go-testutils-toolkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestAssertHasObject(t *testing.T) {
	t.Run("When the object exists", func(t *testing.T) {
		ft := &testutils.FakeTest{}
		c := fake.NewClientBuilder().WithObjects(&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-namespace",
			},
		}).Build()
		assert.True(t, k8stestutils.AssertHasObject(
			ft,
			c,
			&v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-namespace",
				},
			},
			"namespace my-namespace should exist",
		))
		assert.Len(t, ft.ErrorMessages, 0)
		assert.False(t, ft.Failed)
		t.Run("the object data should be populated after the check", func(t *testing.T) {
			ft := &testutils.FakeTest{}
			o := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-ns",
					Name:      "my-cm",
					Labels: map[string]string{
						"my": "label",
					},
				},
				Data: map[string]string{
					"some": "value",
				},
			}
			c := fake.NewClientBuilder().WithObjects(o).Build()
			testedObject := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-ns",
					Name:      "my-cm",
				},
			}
			assert.True(t, k8stestutils.AssertHasObject(
				ft,
				c,
				testedObject,
			))
			assert.Len(t, ft.ErrorMessages, 0)
			assert.False(t, ft.Failed)
			assert.EqualValues(t, o.ObjectMeta, testedObject.ObjectMeta)
			assert.EqualValues(t, o.Data, testedObject.Data)
		})
	})
	t.Run("When the object does not exist", func(t *testing.T) {
		ft := &testutils.FakeTest{}
		c := fake.NewClientBuilder().Build()
		assert.False(t, k8stestutils.AssertHasObject(
			ft,
			c,
			&v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-namespace",
				},
			},
			"namespace my-namespace should exist",
		))
		assert.False(t, ft.Failed)
		require.Len(t, ft.ErrorMessages, 1)
		assert.Contains(t, ft.ErrorMessages[0], `namespaces "my-namespace" not found`)
		assert.Contains(t, ft.ErrorMessages[0], `namespace my-namespace should exist`)
	})
}

func TestRequireHasObject(t *testing.T) {
	t.Run("When the object exists", func(t *testing.T) {
		ft := &testutils.FakeTest{}
		c := fake.NewClientBuilder().WithObjects(&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-namespace",
			},
		}).Build()
		k8stestutils.RequireHasObject(
			ft,
			c,
			&v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-namespace",
				},
			},
			"namespace my-namespace should exist",
		)
		assert.Len(t, ft.ErrorMessages, 0)
		assert.False(t, ft.Failed)
	})
	t.Run("When the object does not exist", func(t *testing.T) {
		ft := &testutils.FakeTest{}
		c := fake.NewClientBuilder().Build()
		k8stestutils.RequireHasObject(
			ft,
			c,
			&v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-namespace",
				},
			},
			"namespace my-namespace should exist",
		)
		assert.True(t, ft.Failed)
		require.Len(t, ft.ErrorMessages, 1)
		assert.Contains(t, ft.ErrorMessages[0], `namespaces "my-namespace" not found`)
		assert.Contains(t, ft.ErrorMessages[0], `namespace my-namespace should exist`)
	})
}

func TestAssertHasNoObject(t *testing.T) {
	t.Run("When the object exists", func(t *testing.T) {
		ft := &testutils.FakeTest{}
		c := fake.NewClientBuilder().WithObjects(&v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "my-namespace",
				Name:      "my-cm",
			},
		}).Build()
		assert.False(t, k8stestutils.AssertHasNoObject(
			ft,
			c,
			&v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-namespace",
					Name:      "my-cm",
				},
			},
			"configmap my-cm should not exist",
		))
		require.Len(t, ft.ErrorMessages, 1)
		assert.False(t, ft.Failed)
		assert.Contains(t, ft.ErrorMessages[0], `Object /v1, Kind=ConfigMap my-namespace/my-cm exists`)
		assert.Contains(t, ft.ErrorMessages[0], `configmap my-cm should not exist`)
	})
	t.Run("When the object does not exist", func(t *testing.T) {
		ft := &testutils.FakeTest{}
		c := fake.NewClientBuilder().Build()
		assert.True(t, k8stestutils.AssertHasNoObject(
			ft,
			c,
			&v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-namespace",
				},
			},
		))
		assert.False(t, ft.Failed)
		assert.Len(t, ft.ErrorMessages, 0)
	})
}
