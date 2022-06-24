package agent

import (
	"context"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// ClusterKubeAPIClient is a kube client to interact with the cluster that agent installer
// is installing.
type ClusterKubeAPIClient struct {
	Client     *kubernetes.Clientset
	ctx        context.Context
	config     *rest.Config
	configPath string
}

// NewClusterKubeAPIClient Create a new kube client to interact with the cluster under install.
func NewClusterKubeAPIClient(ctx context.Context, assetDir string) (*ClusterKubeAPIClient, error) {

	kubeClient := &ClusterKubeAPIClient{}

	kubeconfigpath := filepath.Join(assetDir, "auth", "kubeconfig")
	kubeconfig, err := clientcmd.BuildConfigFromFlags("", kubeconfigpath)
	if err != nil {
		return nil, errors.Wrap(err, "Error loading kubeconfig from assets.")
	}

	kubeclient, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return nil, errors.Wrap(err, "Creating a Kubernetes client from assets failed.")
	}

	kubeClient.Client = kubeclient
	kubeClient.ctx = ctx
	kubeClient.config = kubeconfig
	kubeClient.configPath = kubeconfigpath

	return kubeClient, nil
}

// ClusterKubeAPIClient.IsKubeAPILive Determine if the cluster under install has initailized the
// kubenertes API.
func (kube *ClusterKubeAPIClient) IsKubeAPILive() (bool, error) {

	discovery := kube.Client.Discovery()
	version, err := discovery.ServerVersion()
	if err != nil {
		return false, err
	}
	logrus.Infof("Cluster API is up and running %s", version)
	return true, nil
}

// DoesKubeConfigExist.DoesKubeConfigExist Determine if the kubeconfig for the cluster
// can be used without errors.
func (kube *ClusterKubeAPIClient) DoesKubeConfigExist() (bool, error) {

	_, err := clientcmd.LoadFromFile(kube.configPath)
	if err != nil {
		return false, errors.Wrap(err, "Error loading kubeconfig from file.")
	}
	return true, nil
}

// ClusterKubeAPIClient.IsBootstrapConfigMapComplete Detemine if the cluster's bootstrap
// configmap has the status complete.
func (kube *ClusterKubeAPIClient) IsBootstrapConfigMapComplete() (bool, error) {

	// Get latest version of bootstrap configmap
	bootstrap, err := kube.Client.CoreV1().ConfigMaps("kube-system").Get(kube.ctx, "bootstrap", v1.GetOptions{})

	if err != nil {
		logrus.Debug("bootstrap configmap not found")
		return false, err
	}
	// Found a bootstrap configmap need to check its status
	if bootstrap != nil && err == nil {
		status, ok := bootstrap.Data["status"]
		if !ok {
			logrus.Debug("No status found in bootstrap configmap.")
			return false, nil
		}
		if status == "complete" {
			return true, nil
		}
	}
	return false, nil
}
