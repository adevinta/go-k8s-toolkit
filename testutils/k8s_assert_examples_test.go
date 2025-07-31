package testutils_test

import (
	"context"
	"fmt"
	"testing"

	k8stestutils "github.com/adevinta/go-k8s-toolkit/testutils"
	testutils "github.com/adevinta/go-testutils-toolkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func ExampleAssertHasNoObject() {
	// use the real *testing.T from the test
	t := &testutils.FakeTest{Name: "TestMyObjectDoesNotExist"}

	client := fake.NewClientBuilder().Build()
	k8stestutils.AssertHasNoObject(t, client, &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-namespace",
		},
	})

	// Usually, this is done by the go framework
	fmt.Println(t)
	// Output: --- PASS: TestMyObjectDoesNotExist
}

func ExampleRequireHasNoObject() {
	// use the real *testing.T from the test
	t := &testutils.FakeTest{Name: "TestMyObjectDoesNotExist"}

	client := fake.NewClientBuilder().WithObjects(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-namespace",
		},
	}).Build()
	k8stestutils.RequireHasNoObject(t, client, &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-namespace",
		},
	})

	// Usually, this is done by the go framework
	fmt.Println(t)
	// Output: --- FAIL: TestMyObjectDoesNotExist
}

func ExampleAssertHasObject() {
	// use the real *testing.T from the test
	t := &testutils.FakeTest{Name: "TestMyObjectExists"}

	client := fake.NewClientBuilder().Build()
	k8stestutils.AssertHasObject(t, client, &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-namespace",
		},
	})

	// Usually, this is done by the go framework
	fmt.Println(t)
	// Output: --- FAIL: TestMyObjectExists
}

func ExampleRequireHasObject() {
	// use the real *testing.T from the test
	t := &testutils.FakeTest{Name: "TestMyObjectExists"}

	client := fake.NewClientBuilder().WithObjects(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-namespace",
		},
	}).Build()
	k8stestutils.RequireHasObject(t, client, &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-namespace",
		},
	})

	// Usually, this is done by the go framework
	fmt.Println(t)
	// Output: --- PASS: TestMyObjectExists
}

// start ReadMe examples

func TestMyObjectDoesNotExist(t *testing.T) {
	client := fake.NewClientBuilder().Build()
	k8stestutils.AssertHasNoObject(t, client, &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-namespace",
		},
	})
}

func TestMyObjectCanBeCreated(t *testing.T) {
	client := fake.NewClientBuilder().Build()

	// later
	k8stestutils.RequireHasNoObject(t, client, &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-namespace",
		},
	})

	assert.NoError(t,
		client.Create(context.Background(),
			&v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-namespace",
				},
			},
		),
	)
}

func TestMyObjectExists(t *testing.T) {
	client := fake.NewClientBuilder().Build()

	require.NoError(t,
		client.Create(context.Background(),
			&v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-namespace",
				},
			},
		),
	)

	k8stestutils.AssertHasObject(t, client, &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-namespace",
		},
	})
}

func TestMyObjectCanBeDeleted(t *testing.T) {

	client := fake.NewClientBuilder().WithObjects(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-namespace",
		},
	}).Build()
	k8stestutils.RequireHasObject(t, client, &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-namespace",
		},
	})

	require.NoError(t,
		client.Delete(context.Background(),
			&v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-namespace",
				},
			},
		),
	)
}
