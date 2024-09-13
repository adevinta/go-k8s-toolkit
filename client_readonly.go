package k8s

import (
	"context"
	"errors"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func WithErrorBuilder(newError func(string) error) func(c *readOnlyClient) {
	return func(c *readOnlyClient) {
		c.newError = newError
	}
}

func WithNoError() func(c *readOnlyClient) {
	return WithErrorBuilder(func(string) error {
		return nil
	})
}
func WithError() func(c *readOnlyClient) {
	return WithErrorBuilder(func(method string) error {
		return fmt.Errorf("%s not allowed in read-only mode", method)
	})
}

func ReadOnlyClient(client client.Client, mutators ...func(c *readOnlyClient)) client.Client {
	c := &readOnlyClient{
		Client: client,
		newError: func(method string) error {
			return fmt.Errorf("%s not allowed in read-only mode", method)
		},
	}
	for _, m := range mutators {
		m(c)
	}
	return c
}

type readOnlyClient struct {
	client.Client
	newError func(method string) error
}

type readOnlySubresourceClient struct {
	client.SubResourceClient
	newError func(method string) error
}

var _ client.SubResourceClient = &readOnlySubresourceClient{}

// client must implement interface client.Client
var _ client.Client = &readOnlyClient{}

func (r *readOnlyClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	if r == nil {
		return errors.New("client is nil")
	}
	return r.newError("Create")
}

func (r *readOnlyClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	if r == nil {
		return errors.New("client is nil")
	}
	return r.newError("Update")
}

func (r *readOnlyClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	if r == nil {
		return errors.New("client is nil")
	}
	return r.newError("Patch")
}

func (r *readOnlyClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	if r == nil {
		return errors.New("client is nil")
	}
	return r.newError("Delete")
}

func (r *readOnlyClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	if r == nil {
		return errors.New("client is nil")
	}
	return r.newError("DeleteAllOf")
}

func (r *readOnlyClient) SubResource(resource string) client.SubResourceClient {
	var subResourceClient client.SubResourceClient
	if r != nil && r.Client != nil {
		subResourceClient = r.Client.SubResource(resource)
	}
	return &readOnlySubresourceClient{
		SubResourceClient: subResourceClient,
		newError:          r.newError,
	}
}

func (r *readOnlyClient) Status() client.StatusWriter {
	return r.SubResource("status")
}

func (r *readOnlySubresourceClient) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	if r == nil {
		return errors.New("status client is nil")
	}
	return r.newError("Update")
}
func (r *readOnlySubresourceClient) Create(ctx context.Context, obj client.Object, subResource client.Object, opts ...client.SubResourceCreateOption) error {
	if r == nil {
		return errors.New("status client is nil")
	}
	return r.newError("Update")
}
func (r *readOnlySubresourceClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
	if r == nil {
		return errors.New("status client is nil")
	}
	return r.newError("Update")
}
