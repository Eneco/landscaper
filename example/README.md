## Prerequisites

This example assumes a built Landscaper, a local Helm setup and a kubectl configured to a working Kubernetes cluster.

## Serve Example Charts

The example landscape deploys a `hello-world` Chart. This Chart echoes a specified message to `stdout` every `n` seconds.

From a terminal, locally serve it with:

    cd $GOPATH/src/github.com/eneco/landscaper/example/charts
    helm lint */ && helm package */
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

## Using Secrets

Landscaper can manage secrets. For a component that has secrets, a companion Kubernetes Secret object is created, with the same name as the Helm release. The keys inside the secret object are specified in the `secrets` section of a landscape file. The values of the secret are obtained from the environment the landscaper runs in, looking for an environment variable name that is uppercase and hyphens are replaced with underscores.

For example, consider the `secrets` section of [secretive.yaml](landscape-complex/secretive.yaml):

```yaml
secrets:
  - hello-name
  - hello-age
 ```

For this section, Landscaper creates an object with then name of the Release, and with two keys `hello-name` and `hello-age` that are set to the values from environment variables `HELLO_NAME` and `HELLO_AGE` respectively. In the Chart, the secret values are obtained as follows:

```yaml
secretKeyRef:
  name: {{ .Values.secretsRef }}
  key: hello-name
```

Here, `secretsRef` is filled in by Landscaper with the name of the Secret object, which is identical to the Helm Release.

To make things a bit more concrete, run:

    HELLO_NAME=rumpelstiltskin HELLO_AGE=42 ./build/landscaper apply --dir build --namespace example --dir example/landscape-complex

Upon success, `kubectl get secret --namespace example` should contain a Secret object named `example-secretive`. Use `kubectl get pods --namespace example` to obtain the exact pod name, and inspecting its output with `kubectl --namespace example logs example-secretive-hello-secret-EXACT-NAME` should give output similar to:

    Tue Dec 20 08:34:10 UTC 2016 | my name is rumpelstiltskin and I am 42 years old

