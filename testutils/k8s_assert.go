package testutils

import (
	"context"
	"fmt"

	"github.com/adevinta/go-testutils-toolkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func AssertHasObject(t assert.TestingT, c client.Client, o client.Object, msgAndArgs ...interface{}) bool {
	if h, ok := t.(testutils.TestHelper); ok {
		h.Helper()
	}
	err := c.Get(context.Background(), client.ObjectKeyFromObject(o), o)
	success := assert.NoError(t, err, msgAndArgs...)
	return success
}

func RequireHasObject(t require.TestingT, c client.Client, o client.Object, msgAndArgs ...interface{}) {
	if h, ok := t.(testutils.TestHelper); ok {
		h.Helper()
	}
	if AssertHasObject(
		t,
		c,
		o,
		msgAndArgs...) {
		return
	}
	t.FailNow()
}

func AssertHasNoObject(t assert.TestingT, c client.Client, o client.Object, msgAndArgs ...interface{}) bool {
	if h, ok := t.(testutils.TestHelper); ok {
		h.Helper()
	}
	err := c.Get(context.Background(), client.ObjectKeyFromObject(o), o)
	success := true
	if err != nil {
		success = assert.NoError(t, client.IgnoreNotFound(err), msgAndArgs...)
	} else {
		gvks, _, err := c.Scheme().ObjectKinds(o)
		if err != nil || len(gvks) == 0 {
			success = assert.Fail(t, fmt.Sprintf("Unknown object kind %T", o), msgAndArgs...)
		} else {
			success = assert.Fail(t, fmt.Sprintf("Object %v %v exists", gvks[0].String(), client.ObjectKeyFromObject(o)), msgAndArgs...)
		}
	}
	return success
}

func RequireHasNoObject(t require.TestingT, c client.Client, o client.Object, msgAndArgs ...interface{}) {
	if h, ok := t.(testutils.TestHelper); ok {
		h.Helper()
	}
	if AssertHasNoObject(
		t,
		c,
		o,
		msgAndArgs...) {
		return
	}
	t.FailNow()
}
