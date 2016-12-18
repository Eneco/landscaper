## Prerequisites

This example assumes a built Landscaper, a local Helm setup and a kubectl configured to a working Kubernetes cluster.

## Serve Example Charts

The example landscape deploys a `hello-world` Chart. This Chart echoes a specified message to `stdout` every `n` seconds.

From a terminal, locally serve it with:

	cd $GOPATH/src/github.com/eneco/landscaper/example/charts
	helm lint hello-world && helm package hello-world
	helm serve

## Apply an Example Landscape

The example landscape deploys two instances of `hello-world`.
[landscape/hello-kubernetes.yaml](One) greets Kubernetes people; the [landscape/hello-world.yaml](other) welcomes a landscaped world.

From another terminal, dry-run the landscape:

	cd $GOPATH/src/github.com/eneco/landscaper
	./build/landscaper apply --dir example/landscape --namespace example --dry-run

When happy with the output of the Landscaper, deploy the `hello-worlds` for real by omitting `--dry-run`:

	./build/landscaper apply --dir example/landscape --namespace example

To inspect the deployment:

	kubectl get pods --namespace example

Output should be similar to:

	NAME                                                    READY     STATUS    RESTARTS   AGE
	example-hello-kubernetes-hello-world-2511913442-n29c8   1/1       Running   0          25s
	example-hello-world-hello-world-3610099532-w0shn        1/1       Running   0          25s

Inspect the output of the deployments with:

	kubectl logs --namespace example <one pod>
	kubectl logs --namespace example <the other pod>

But wait! `hello-kubernetes` contained a typo! Change the message yourself by editting `hello-kubernetes.yaml` or use the fixed landscape provided in `example/landscape-fixed`:

 	./build/landscaper apply --dir example/landscape-fixed --namespace example
