package k8s_test

import (
	"context"
	"os"
	"strings"
	"testing"

	k8s "github.com/adevinta/go-k8s-toolkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestKind(t *testing.T) {
	kind := k8s.KinDForVersion("v1.15.3")
	cluster, err := kind.Start("kind-test", "v1.15.3")
	require.NoError(t, err)
	assert.Equal(t, ".kind/.kube/config-kind-test-v1.15.3", cluster.KubeConfigPath())
	cfg, err := k8s.NewClientConfigBuilder().WithKubeConfigPath(cluster.KubeConfigPath()).Build()
	assert.NoError(t, err)
	client, err := k8sclient.New(cfg, k8sclient.Options{})
	assert.NoError(t, err)
	pods := v1.PodList{}
	assert.NoError(t, client.List(context.Background(), &pods))
	expectedPods := map[string]interface{}{
		"coredns":                 nil,
		"etcd":                    nil,
		"kube-apiserver":          nil,
		"kube-controller-manager": nil,
		"kube-proxy":              nil,
		"kube-scheduler":          nil,
	}
	for _, pod := range pods.Items {
		for name := range expectedPods {
			if strings.HasPrefix(pod.Name, name) {
				delete(expectedPods, name)
				assert.Equal(t, v1.PodRunning, pod.Status.Phase)
				break
			}
		}
	}
	assert.Len(t, expectedPods, 0)
	assert.NoError(t, kind.Delete(cluster))
	assert.Error(t, client.List(context.Background(), &pods))
	_, err = os.Stat(cluster.KubeConfigPath())
	assert.True(t, os.IsNotExist(err))
}
