package landscaper

import (
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	restclient "k8s.io/client-go/rest"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/kube"
	helmversion "k8s.io/helm/pkg/version"
	podutil "k8s.io/kubernetes/pkg/api/pod"
	"k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/typed/core/internalversion"
)

// TODO refactor out this global var
var tillerTunnel *kube.Tunnel

// Environment contains all the information about the k8s cluster and local configuration
type Environment struct {
	HelmHome                  string        // Helm's home directory
	DryRun                    bool          // If true, don't modify anything
	Wait                      bool          // Wait until all resources become ready
	WaitTimeout               time.Duration // Wait for helm
	ChartLoader               ChartLoader   // ChartLoader loads charts
	ReleaseNamePrefix         string        // Prepend this string to release names
	ComponentFiles            []string      // Landscaper component file names
	LandscapeDir              string        // deprecated: ComponentFiles is leading; LandscapeDir merely fills it
	Namespace                 string        // Default namespace releases are put into; components can override it though
	Verbose                   bool          // Reduce log level
	Context                   string        // Kubernetes context to use
	Loop                      bool          // Keep looping
	LoopInterval              time.Duration // Loop every duration
	TillerNamespace           string        // where to install / use tiller
	AzureKeyVault             string        // Azure keyvault to use for secrets if provided
	Environment               string        // Environment selections
	ConfigurationOverrideFile string        // Global configuration overrides file
	helmClient                helm.Interface
	kubeClient                internalversion.CoreInterface
	DisabledStages            stringSlice // stages to disable during landscaper apply
}

type stringSlice []string

func (stringSlice *stringSlice) String() string {
	return ""
}

func (stringSlice *stringSlice) Set(value string) error {
	*stringSlice = append(*stringSlice, value)
	return nil
}

func (stringSlice *stringSlice) Type() string {
	return "stringSlice"
}

// HelmClient makes sure the environment has a HelmClient initialized and returns it
func (e *Environment) HelmClient() helm.Interface {
	if e.helmClient == nil {
		logrus.WithFields(logrus.Fields{"helmClientVersion": helmversion.Version, "tillerNamespace": e.TillerNamespace}).Debug("Setup Helm Client")

		tillerHost, err := setupConnection(e.Context, e.TillerNamespace)
		if err != nil {
			logrus.WithField("error", err).Fatalf("Could not set up connection to helm")
			return nil
		}

		logrus.WithField("host", tillerHost).Debug("Tiller host address")

		e.helmClient = helm.NewClient(helm.Host(tillerHost), helm.ConnectTimeout(5))
		logrus.WithField("client", e.helmClient).Debug("Helm client")

		err = e.helmClient.PingTiller()

		if err != nil {
			logrus.WithField("error", err).Fatalf("Can't ping Tiller")
		}

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
func (e *Environment) KubeClient() internalversion.CoreInterface {
	if e.kubeClient == nil {
		logrus.Debug("Setup Kubernetes Client")

		_, client, err := getKubeClient(e.Context)
		if err != nil {
			logrus.WithField("error", err).Fatalf("Could not build Kubernetes client config")
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

// setupConnection creates and returns tiller port forwarding tunnel
func setupConnection(context string, tillerNamespace string) (string, error) {
	helmHost, helmHostExists := os.LookupEnv("HELM_HOST")
	if helmHostExists {
		logrus.WithFields(logrus.Fields{"helmHost": helmHost}).Debug("Using tiller address from HELM_HOST")
		return helmHost, nil
	}

	logrus.WithFields(logrus.Fields{"tillerNamespace": tillerNamespace}).Debug("Create tiller tunnel")
	tunnel, err := newTillerPortForwarder(tillerNamespace, context)
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
func getKubeClient(context string) (*restclient.Config, *internalclientset.Clientset, error) {
	config, err := kube.GetConfig(context).ClientConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("could not get kubernetes config for context '%s': %s", context, err)
	}
	client, err := internalclientset.NewForConfig(config)
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

	podName, err := getTillerPodName(client.Core(), namespace)
	if err != nil {
		return nil, err
	}
	const tillerPort = 44134
	t := kube.NewTunnel(client.Core().RESTClient(), config, namespace, podName, tillerPort)
	return t, t.ForwardPort()
}

func getTillerPodName(client internalversion.PodsGetter, namespace string) (string, error) {
	// TODO use a const for labels
	selector := labels.Set{"app": "helm", "name": "tiller"}.AsSelector()
	pod, err := getFirstRunningPod(client, namespace, selector)
	if err != nil {
		return "", err
	}
	return pod.ObjectMeta.GetName(), nil
}

func getFirstRunningPod(client internalversion.PodsGetter, namespace string, selector labels.Selector) (*core.Pod, error) {
	options := metav1.ListOptions{LabelSelector: selector.String()}
	pods, err := client.Pods(namespace).List(options)
	if err != nil {
		return nil, err
	}
	if len(pods.Items) < 1 {
		return nil, fmt.Errorf("could not find tiller")
	}
	for _, p := range pods.Items {
		if podutil.IsPodReady(&p) {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("could not find a ready tiller pod")
}
