package k8s

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const DefaultVersion = "v0.11.1"

type KubeCluster interface {
	KubeConfigPath() string
}

type KinD struct {
	Dir     string
	Version string
}

type KinDCluster struct {
	dir     string
	name    string
	version string
	kind    *KinD
}

var DefaultKind = KinD{
	Dir:     "./.kind",
	Version: DefaultVersion,
}

func (k *KinD) ListClusters() []string {
	c := exec.Command(k.path(), "get", "clusters")
	b := &bytes.Buffer{}
	c.Stdout = b
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return []string{}
	}
	r := strings.Split(b.String(), "\n")
	for i, s := range r {
		r[i] = strings.Trim(s, " \n")
	}
	return r
}

func (k *KinD) Start(name, version string) (*KinDCluster, error) {
	_, err := os.Stat(k.path())
	if err != nil {
		if err := k.Install(); err != nil {
			return nil, err
		}
	}
	cluster := &KinDCluster{
		dir:     k.Dir,
		name:    name,
		version: version,
		kind:    k,
	}
	os.Setenv("KUBECONFIG", cluster.KubeConfigPath())
	if !cluster.Exists() {
		err := os.MkdirAll(filepath.Dir(cluster.KubeConfigPath()), 0777)
		if err != nil {
			return nil, err
		}
		args := []string{"create", "cluster", "--image", "kindest/node:" + version, "--name", cluster.ID()}
		if k.Version != "v0.5.0" {
			args = append(args, "--kubeconfig", cluster.KubeConfigPath())
		} else {
			os.Remove(cluster.KubeConfigPath())
		}
		c := exec.Command(k.path(), args...)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		err = c.Run()
		if err != nil {
			dir, _ := ioutil.TempDir("", "example")
			if err != nil {
				return nil, err
			}
			defer os.RemoveAll(dir)

			c := exec.Command(k.path(), "export", "logs", dir, "--name", cluster.ID())
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			c.Run()
			filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				if info.IsDir() {
					return nil
				}
				fmt.Println("######", path)
				fd, err := os.Open(path)
				if err != nil {
					return err
				}
				defer fd.Close()
				_, err = io.Copy(os.Stdout, fd)
				return err
			})
			return nil, err
		}
	}
	err = cluster.DownloadKubeConfig()
	if err != nil {
		return cluster, err
	}
	for {
		cfg, err := NewClientConfigBuilder().WithKubeConfigPath(cluster.KubeConfigPath()).Build()
		if err != nil {
			return nil, err
		}
		client, err := k8sclient.New(cfg, k8sclient.Options{})
		if err != nil {
			return nil, err
		}
		pods := v1.PodList{}
		if err = client.List(context.Background(), &pods); err == nil {
			if len(pods.Items) >= 8 {
				// all required pods seems to be there, checking they are ready
				initialized := true
				for _, p := range pods.Items {
					if p.Status.Phase != "Running" {
						initialized = false
					}
				}
				if initialized {
					break
				}
			}
		}
		fmt.Println("cluster is still initializing, waiting a bit")
		time.Sleep(500 * time.Millisecond)
	}
	return cluster, nil
}

func (k *KinD) Delete(cluster *KinDCluster) error {
	c := exec.Command(k.path(), "delete", "cluster", "--name", cluster.ID())
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	err := c.Run()
	if err != nil {
		return err
	}
	return os.Remove(cluster.KubeConfigPath())
}

func (k *KinD) Install() error {
	// map linux (GOOS) to Linux (result of uname), darwin (GOOS) to Darwin (result of uname)
	resp, err := http.Get(fmt.Sprintf("https://kind.sigs.k8s.io/dl/%s/kind-%s-%s", k.Version, strings.Title(runtime.GOOS), runtime.GOARCH))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	err = os.MkdirAll(filepath.Dir(k.path()), 0777)
	if err != nil {
		return err
	}
	fd, err := os.OpenFile(k.path(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		return err
	}
	defer fd.Close()
	_, err = io.Copy(fd, resp.Body)
	return err
}

func (k *KinD) Exists(name string) bool {
	for _, cluster := range k.ListClusters() {
		if cluster == name {
			return true
		}
	}
	return false
}

func (k *KinD) DownloadKubeConfig(name string) (string, error) {
	c := exec.Command(k.path(), "get", "kubeconfig", "--name", name)
	b := &bytes.Buffer{}
	c.Stdout = b
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return "", err
	}
	return b.String(), nil
}
func (k *KinD) path() string {
	return filepath.Join(k.Dir, "bin", "kind-"+k.Version)
}

func (k *KinDCluster) DownloadKubeConfig() error {
	_, err := os.Stat(k.KubeConfigPath())
	if err == nil {
		return nil
	}
	config, err := k.kind.DownloadKubeConfig(k.ID())
	if err != nil {
		return err
	}
	fd, err := os.Create(k.KubeConfigPath())
	if err != nil {
		return err
	}
	defer fd.Close()
	_, err = fd.WriteString(config)
	return err
}

func (k *KinDCluster) ID() string {
	return k.name + "-" + k.version
}
func (k *KinDCluster) Exists() bool {
	return k.kind.Exists(k.ID())
}

func (k *KinDCluster) KubeConfigPath() string {
	return filepath.Join(k.dir, ".kube", "config-"+k.ID())
}

func KinDForVersion(version string) *KinD {
	return &DefaultKind
}
