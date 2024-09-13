package k8s_test

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"

	k8s "github.com/adevinta/go-k8s-toolkit"
	system "github.com/adevinta/go-system-toolkit"
	testutils "github.com/adevinta/go-testutils-toolkit"
	"github.com/google/uuid"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKubeConfigPath(t *testing.T) {
	t.Cleanup(system.Reset)
	os.Unsetenv("KUBECONFIG")
	os.Setenv("HOME", "./no-home")

	assert.Equal(t, "", k8s.KubeConfigPath(""))
	os.Setenv("HOME", "./test-data/home")
	assert.Equal(t, "test-data/home/.kube/config", k8s.KubeConfigPath(""))

	os.Setenv("HOME", "./no-home")
	os.Setenv("KUBECONFIG", "./test-data/home/.kube/config")
	assert.Equal(t, "test-data/home/.kube/config", k8s.KubeConfigPath(""))

	os.Setenv("HOME", "./no-home")
	os.Unsetenv("KUBECONFIG")
	assert.Equal(t, "test-data/home/.kube/config", k8s.KubeConfigPath("./test-data/home/.kube/config"))
}

func TestImpersonateUserName(t *testing.T) {
	builder := k8s.NewClientConfigBuilder()
	builder.WithKubeConfigPath("./test-data/home/.kube/config")
	builder = builder.WithImpersonateUserName("test-user")
	config, err := builder.Build()
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, "test-user", config.Impersonate.UserName)
}

func TestImpersonateGroups(t *testing.T) {
	t.Run("without impersonate username an error is returned", func(t *testing.T) {
		builder := k8s.NewClientConfigBuilder()
		builder.WithKubeConfigPath("./test-data/home/.kube/config")
		builder = builder.WithImpersonateUserGroups("test-group", "test-groups-2")
		_, err := builder.Build()
		assert.Errorf(t, err, "impersonate group without a user should be reported as an error. Kubernetes does not allow it")
	})
	t.Run("with impersonate groups is configured", func(t *testing.T) {
		builder := k8s.NewClientConfigBuilder()
		builder.WithKubeConfigPath("./test-data/home/.kube/config")
		builder = builder.WithImpersonateUserName("test-user")
		builder = builder.WithImpersonateUserGroups("test-group", "test-groups-2")
		_, err := builder.Build()
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"test-group", "test-groups-2"}, builder.ConfigOverrides.AuthInfo.ImpersonateGroups)
	})
}

func TestClientConfigBuilder(t *testing.T) {
	t.Run("When not in github actions", func(t *testing.T) {
		t.Run("When a kubeconfig is available", func(t *testing.T) {
			kubeconfigPath := fmt.Sprintf("./kubeconfig.%s", uuid.New().String())
			t.Cleanup(system.Reset)
			t.Cleanup(func() { os.Remove(kubeconfigPath) })
			os.Unsetenv("ACTIONS_ID_TOKEN_REQUEST_URL")
			os.Unsetenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN")
			http.DefaultTransport = testutils.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
				return nil, errors.New("unexpected http call to " + r.URL.String())
			})

			testutils.EnsureYAMLFileContent(t, system.DefaultFileSystem, kubeconfigPath, map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Config",
				"users": []interface{}{
					map[string]interface{}{
						"name": "user-name",
						"user": map[string]string{
							"token": "k8s-token",
						},
					},
				},
				"current-context": "test",
				"contexts": []interface{}{
					map[string]interface{}{
						"name": "test",
						"context": map[string]string{
							"cluster": "cluster-name",
							"user":    "user-name",
						},
					},
				},
				"clusters": []interface{}{

					map[string]interface{}{
						"name": "cluster-name",
						"cluster": map[string]string{
							"server": "https://k8s.tld",
						},
					},
				},
			})
			cfg, err := k8s.NewClientConfigBuilder().WithKubeConfigPath(kubeconfigPath).Build()
			assert.NoError(t, err)
			require.NotNil(t, cfg)
			assert.Equal(t, "k8s-token", cfg.BearerToken)
		})
		t.Run("When no kubeconfig is available", func(t *testing.T) {
			t.Cleanup(system.Reset)
			os.Unsetenv("ACTIONS_ID_TOKEN_REQUEST_URL")
			os.Unsetenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN")
			system.DefaultFileSystem = afero.NewMemMapFs()
			// github actions runs in k8s and does not clean k8s environment variables
			// This makes the integration consider being in-cluster by default
			os.Unsetenv("KUBERNETES_SERVICE_HOST")
			http.DefaultTransport = testutils.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
				return nil, errors.New("unexpected http call to " + r.URL.String())
			})
			cfg, err := k8s.NewClientConfigBuilder().Build()
			assert.Error(t, err)
			assert.Nil(t, cfg)
		})
	})
}
