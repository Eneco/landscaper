package landscaper

import (
	"fmt"
	"strings"

	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/kube"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/labels"
)

// TODO refactor out this global var
var tillerTunnel *kube.Tunnel
var tillerNamespace = "kube-system"

type Environment struct {
	Name           string
	Namespace      string
	RepositoryName string
	StateFolder    string
	HelmClient     *helm.Client
}

func (e *Environment) EnsureHelmClient() error {
	if e.HelmClient == nil {
		tillerHost, err := setupConnection()
		if err != nil {
			return err
		}

		e.HelmClient = helm.NewClient(helm.Host(tillerHost))
	}

	return nil
}

// ReleaseName taks a component name, and uses info in the environment to return a release name
func (e *Environment) ReleaseName(componentName string) string {
	return fmt.Sprintf("%s-%s", strings.ToLower(string(e.Name[0])), strings.ToLower(componentName))
}

func setupConnection() (string, error) {
	tunnel, err := newTillerPortForwarder(tillerNamespace, "")
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(":%d", tunnel.Local), nil
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
