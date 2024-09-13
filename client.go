package k8s

import (
	"errors"
	"os"
	"path/filepath"

	system "github.com/adevinta/go-system-toolkit"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func KubeConfigPath(configPath string) string {
	for _, dflt := range []string{os.Getenv("KUBECONFIG"), filepath.Join(os.Getenv("HOME"), ".kube", "config")} {
		if configPath == "" {
			configPath = dflt
		}
	}
	_, err := system.DefaultFileSystem.Stat(configPath)
	if err != nil {
		return ""
	}
	return filepath.Clean(configPath)
}

type ClientConfigBuilder struct {
	ClientConfigLoadingRules *clientcmd.ClientConfigLoadingRules
	ConfigOverrides          *clientcmd.ConfigOverrides
	DefaultServerURL         string
	tokenFile                string
}

// NewClientConfigBuilder allows the creation of a flexible Kubernetes client configuration
// creator
func NewClientConfigBuilder() ClientConfigBuilder {
	return ClientConfigBuilder{
		ClientConfigLoadingRules: &clientcmd.ClientConfigLoadingRules{},
		ConfigOverrides:          &clientcmd.ConfigOverrides{},
	}
}

// WithServerURL forces the Kubernetes server URL regardless of the kubeconfig content
func (b ClientConfigBuilder) WithTokenFile(token string) ClientConfigBuilder {
	b.tokenFile = token
	return b
}

// WithServerURL forces the Kubernetes server URL regardless of the kubeconfig content
func (b ClientConfigBuilder) WithServerURL(url string) ClientConfigBuilder {
	b.ConfigOverrides.ClusterInfo.Server = url
	return b
}

// WithDefaultServerURL allows to fallback to a given Kubernetes server URL in case no config path exist
// or server URL is not provided
func (b ClientConfigBuilder) WithDefaultServerURL(url string) ClientConfigBuilder {
	b.DefaultServerURL = url
	return b
}

// WithKubeConfigPath defines the kubeconfig file path to be loaded.
// If the filepath is empty or does not exist, the client will fallback to the default kubeconfig paths
// pointed by the ${KUBECONFIG} environment variable and ${HOME}/.kube/config
func (b ClientConfigBuilder) WithKubeConfigPath(path string) ClientConfigBuilder {
	b.ClientConfigLoadingRules.ExplicitPath = path
	return b
}

// WithContext allows to define the kubernetes context to use.
// Equivalent to `kubectl --context ${ctx}`
func (b ClientConfigBuilder) WithContext(ctx string) ClientConfigBuilder {
	b.ConfigOverrides.CurrentContext = ctx
	return b
}

// WithImpersonateUserName allows to create a client configuration with impersonation.
// Equivalent to `kubectl --as ${user}`
func (b ClientConfigBuilder) WithImpersonateUserName(userName string) ClientConfigBuilder {
	b.ConfigOverrides.AuthInfo.Impersonate = userName
	return b
}

// WithImpersonateUserGroups allows to create a client configuration with impersonation.
// Equivalent to `kubectl --as my-user --as-group ${group}`
func (b ClientConfigBuilder) WithImpersonateUserGroups(userGroups ...string) ClientConfigBuilder {
	b.ConfigOverrides.AuthInfo.ImpersonateGroups = userGroups
	return b
}

func (b ClientConfigBuilder) populateK8sClientToken(cfg *restclient.Config) error {
	if cfg == nil {
		return errors.New("nil rest config")
	}
	// When there is no authentication in the config, try to discover it
	if cfg.BearerToken == "" && cfg.BearerTokenFile == "" && cfg.TLSClientConfig.KeyFile == "" && len(cfg.TLSClientConfig.KeyData) == 0 && cfg.ExecProvider == nil {
		kubeconfigPath := KubeConfigPath("")
		if kubeconfigPath != "" {
			tokenFile := filepath.Join(filepath.Dir(kubeconfigPath), b.tokenFile)
			token, err := os.ReadFile(tokenFile)
			if err == nil {
				cfg.BearerToken = string(token)
			}
		}
	}
	return nil
}

// Build generates a new rest client config for the current builder.
func (b ClientConfigBuilder) Build() (*restclient.Config, error) {
	cfg := &restclient.Config{}
	var err error
	b.ClientConfigLoadingRules.ExplicitPath = KubeConfigPath(b.ClientConfigLoadingRules.ExplicitPath)

	if b.ConfigOverrides.ClusterInfo.Server == "" && b.ClientConfigLoadingRules.ExplicitPath == "" && b.DefaultServerURL != "" {
		b.ConfigOverrides.ClusterInfo.Server = b.DefaultServerURL
	}

	cfg, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(b.ClientConfigLoadingRules, b.ConfigOverrides).ClientConfig()
	if err != nil {
		return nil, err
	}

	err = b.populateK8sClientToken(cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
