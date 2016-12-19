package landscaper

import (
	"fmt"
	"strings"

	"github.com/Sirupsen/logrus"
	"k8s.io/client-go/1.4/kubernetes"
	v1core "k8s.io/client-go/1.4/kubernetes/typed/core/v1"
	"k8s.io/client-go/1.4/tools/clientcmd"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/kube"
	helmversion "k8s.io/helm/pkg/version"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/labels"
)

// TODO refactor out this global var
var tillerTunnel *kube.Tunnel
var tillerNamespace = "kube-system"

// Environment contains all the information about the k8s cluster and local configuration
type Environment struct {
	ChartDir          string
	DryRun            bool
	ChartLoader       ChartLoader
	ReleaseNamePrefix string
	LandscapeDir      string
	Namespace         string
	Verbose           bool
	NoCronUpdate      bool // NoCronUpdate replaces a CronJob update with a create+delete; k8s #35149 work around

	helmClient helm.Interface
	kubeClient v1core.CoreInterface
}

// HelmClient makes sure the environment has a HelmClient initialized and returns it
func (e *Environment) HelmClient() helm.Interface {
	if e.helmClient == nil {
		logrus.WithFields(logrus.Fields{"helmClientVersion": helmversion.Version}).Debug("Setup Helm Client")

		tillerHost, err := setupConnection()
		if err != nil {
			logrus.WithField("error", err).Fatalf("Could not set up connection to helm")
			return nil
		}

		e.helmClient = helm.NewClient(helm.Host(tillerHost))

		tillerVersion, err := e.helmClient.GetVersion()
		if err != nil {
			logrus.WithField("error", err).Fatalf("Could not retrieve Helm Tiller version")
			return nil
		}

		compatible := helmversion.IsCompatible(helmversion.Version, tillerVersion.Version.SemVer)
		logrus.WithFields(logrus.Fields{"tillerVersion": tillerVersion.Version.SemVer, "clientServerCompatible": compatible}).Info("Connected to Tiller")

		if !compatible {
			logrus.Warn("Helm and Tiller report incompatible version numbers")
		}
	}

	return e.helmClient
}

// KubeClient makes sure the environment has a KubeClient initialized
func (e *Environment) KubeClient() v1core.CoreInterface {
	if e.kubeClient == nil {
		logrus.Debug("Setup Kubernetes Client")

		cfg, err := clientcmd.BuildConfigFromFlags("", clientcmd.NewDefaultPathOptions().GlobalFile)
		if err != nil {
			logrus.WithField("error", err).Fatalf("Could not build Kubernetes client config")
			return nil
		}

		client, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			logrus.WithField("error", err).Fatalf("Could not create Kubernetes client")
			return nil
		}

		version, err := client.ServerVersion()
		if err != nil {
			logrus.WithField("error", err).Fatalf("Could not create retrieve Kubernetes server version")
			return nil
		}

		logrus.WithFields(logrus.Fields{"kubernetesVersion": version.String()}).Info("Connected to Kubernetes")

		e.kubeClient = client.Core()
	}

	return e.kubeClient
}

// Teardown closes the tunnel
func (e *Environment) Teardown() {
	teardown()
}

// ReleaseName takes a component name, and uses info in the environment to return a release name
func (e *Environment) ReleaseName(componentName string) string {
	return e.ReleaseNamePrefix + strings.ToLower(componentName)
}

// setupConnection creates and returns tiller port forwarding tunnel
func setupConnection() (string, error) {
	logrus.WithFields(logrus.Fields{"tillerNamespace": tillerNamespace}).Debug("Create tiller tunnel")
	tunnel, err := newTillerPortForwarder(tillerNamespace, "")
	if err != nil {
		logrus.WithFields(logrus.Fields{"tillerNamespace": tillerNamespace, "error": err}).Error("Failed to create tiller tunnel")
		return "", err
	}

	tillerTunnel = tunnel

	logrus.WithFields(logrus.Fields{"port": tunnel.Local}).Debug("Created tiller tunnel")

	return fmt.Sprintf(":%d", tunnel.Local), nil
}

// teardown closes the tunnel
func teardown() {
	if tillerTunnel != nil {
		logrus.Info("teardown tunnel")
		tillerTunnel.Close()
		tillerTunnel = nil
	}
}

// getKubeClient is a convenience method for creating kubernetes config and client
// for a given kubeconfig context
func getKubeClient(context string) (*restclient.Config, *unversioned.Client, error) {
	config, err := kube.GetConfig(context).ClientConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("could not get kubernetes config for context '%s': %s", context, err)
	}
	client, err := unversioned.New(config)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get kubernetes client: %s", err)
	}
	return config, client, nil
}

func newTillerPortForwarder(namespace, context string) (*kube.Tunnel, error) {
	config, client, err := getKubeClient(context)
	if err != nil {
		return nil, err
	}

	podName, err := getTillerPodName(client, namespace)
	if err != nil {
		return nil, err
	}
	const tillerPort = 44134
	t := kube.NewTunnel(client.RESTClient, config, namespace, podName, tillerPort)
	return t, t.ForwardPort()
}

func getTillerPodName(client unversioned.PodsNamespacer, namespace string) (string, error) {
	// TODO use a const for labels
	selector := labels.Set{"app": "helm", "name": "tiller"}.AsSelector()
	pod, err := getFirstRunningPod(client, namespace, selector)
	if err != nil {
		return "", err
	}
	return pod.ObjectMeta.GetName(), nil
}

func getFirstRunningPod(client unversioned.PodsNamespacer, namespace string, selector labels.Selector) (*api.Pod, error) {
	options := api.ListOptions{LabelSelector: selector}
	pods, err := client.Pods(namespace).List(options)
	if err != nil {
		return nil, err
	}
	if len(pods.Items) < 1 {
		return nil, fmt.Errorf("could not find tiller")
	}
	for _, p := range pods.Items {
		if api.IsPodReady(&p) {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("could not find a ready tiller pod")
}
