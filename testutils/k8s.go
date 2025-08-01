package testutils

import (
	"context"
	"fmt"
	"strings"
	"time"

	k8s "github.com/adevinta/go-k8s-toolkit"
	log "github.com/adevinta/go-log-toolkit"
	"github.com/adevinta/go-testutils-toolkit"
	"github.com/cenkalti/backoff"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	newBackOff = func() backoff.BackOff {
		b := backoff.NewExponentialBackOff()
		b.MaxInterval = 5 * time.Second
		b.MaxElapsedTime = 20 * time.Second
		return b
	}
)

func CreateOrUpdateAll(t require.TestingT, ctx context.Context, c client.Client, objects ...client.Object) {
	if h, ok := t.(testutils.TestHelper); ok {
		h.Helper()
	}
	unstructuredObjects, err := k8s.ToUnstructured(c.Scheme(), objects...)
	require.NoError(t, err)

	// In some occasions, objects need time to be initialised and allow others to be injected (for example for mutating webhooks and deployments)
	// give some time to initialise those
	require.NoError(t, backoff.Retry(
		func() error {
			// There might be inter-dependencies between objects (like CRDs and their objects, roles and role bindings, ...)
			//
			for {
				creationsOrUpdates := 0
				errors := 0
				for _, obj := range unstructuredObjects {
					ref := obj.DeepCopy()
					result, err := ctrl.CreateOrUpdate(ctx, c, ref, func() error {
						uid := ref.GetUID()
						creationTimestamp := ref.GetCreationTimestamp()
						resourceVersion := ref.GetResourceVersion()
						obj.DeepCopyInto(ref)
						ref.SetUID(uid)
						ref.SetCreationTimestamp(creationTimestamp)
						ref.SetResourceVersion(resourceVersion)
						return nil
					})
					if err != nil {
						errors++
						log.DefaultLogger.WithContext(ctx).WithFields(logrus.Fields{"kind": obj.GetKind(), "namespace": obj.GetNamespace(), "name": obj.GetName()}).WithError(err).Trace("failed to create or update object")
					} else {
						if result != controllerutil.OperationResultNone {
							creationsOrUpdates++
						}
					}
				}
				if errors == 0 {
					// no errors: We have applied everything
					return nil
				}
				//
				if creationsOrUpdates == 0 {
					return backoff.Permanent(fmt.Errorf("new retry did not manage to create any new dependent objects. Failing hard"))
				}
			}
		},
		newBackOff(),
	))
}

func FilterUnstructuredObjects(objects []*unstructured.Unstructured, filters ...Filter) []*unstructured.Unstructured {
	r := []*unstructured.Unstructured{}
	for _, o := range objects {
		include := true
		for _, filter := range filters {
			if !filter.Include(o) {
				include = false
				break
			}
			o = filter.Mutate(o)
		}
		if include {
			r = append(r, o)
		}
	}
	return r
}

type Filter interface {
	Include(*unstructured.Unstructured) bool
	Mutate(*unstructured.Unstructured) *unstructured.Unstructured
}

type MutateFilterFunc func(*unstructured.Unstructured) *unstructured.Unstructured

func (f MutateFilterFunc) Mutate(o *unstructured.Unstructured) *unstructured.Unstructured {
	return f(o)
}
func (f MutateFilterFunc) Include(o *unstructured.Unstructured) bool {
	return true
}

type IncludeFilterFunc func(*unstructured.Unstructured) bool

func (f IncludeFilterFunc) Mutate(o *unstructured.Unstructured) *unstructured.Unstructured {
	return o
}
func (f IncludeFilterFunc) Include(o *unstructured.Unstructured) bool {
	return f(o)
}

type ObjectSelector func(object *unstructured.Unstructured) bool

func And(selectors ...ObjectSelector) ObjectSelector {
	return func(object *unstructured.Unstructured) bool {
		for _, s := range selectors {
			if !s(object) {
				return false
			}
		}
		return true
	}
}

func WithKind(kind string) ObjectSelector {
	return func(object *unstructured.Unstructured) bool {
		return object.GetKind() == kind
	}
}

func WithName(name string) ObjectSelector {
	return func(object *unstructured.Unstructured) bool {
		return object.GetName() == name
	}
}

func WithNamespace(namespace string) ObjectSelector {
	return func(object *unstructured.Unstructured) bool {
		return object.GetNamespace() == namespace
	}
}

func ExcludeObject(selector ObjectSelector) Filter {
	return IncludeFilterFunc(func(object *unstructured.Unstructured) bool {
		return !selector(object)
	})
}

func ExtractObjectName(selector ObjectSelector, dest *string) Filter {
	return MutateFilterFunc(func(object *unstructured.Unstructured) *unstructured.Unstructured {
		if selector(object) {
			*dest = object.GetName()
		}
		return object
	})
}
func ExtractObjectNamespace(selector ObjectSelector, dest *string) Filter {
	return MutateFilterFunc(func(object *unstructured.Unstructured) *unstructured.Unstructured {
		if selector(object) {
			*dest = object.GetNamespace()
		}
		return object
	})
}

func ExtractObjectField[T interface{}](t require.TestingT, selector ObjectSelector, dest *T, fields ...string) Filter {
	if h, ok := t.(testutils.TestHelper); ok {
		h.Helper()
	}
	return MutateFilterFunc(func(object *unstructured.Unstructured) *unstructured.Unstructured {
		if selector(object) {
			val, ok, err := unstructured.NestedFieldNoCopy(object.Object, fields...)
			require.NoError(t, err)
			require.True(t, ok, "unable to find field %s", strings.Join(fields, "."))
			switch v := val.(type) {
			case T:
				*dest = v
			default:
				t.Errorf("Unexpected type %T for field %s", v, strings.Join(fields, "."))
				t.FailNow()
			}
		}
		return object
	})
}
