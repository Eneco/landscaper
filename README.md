# Landscaper

Landscaper takes a set of Helm Chart references with values (a desired state), and realizes this in a Kubernetes cluster. The intended use case is to have this desired state under version control, and let Landscaper first test and then apply the state as part of the CI/CD stages.

The Landscaper project is an ongoing project in an early stage, opened due to demand in the community. Contributions and feedback are more than welcome!

## Introduction

### Why

 - The set of applications in the Kubernetes cluster is potentially large and complex;
 - Manually inspecting and administering them, and keeping different environments (e.g. production, acceptance) in sync is laborious and difficult;
 - Cooperating with multiple tenants in this shared cluster increases the complexity.

### How

 - Have a blue print of what the landscape (apps in the cluster) looks like;
 - Keep track of changes: when, what, why and by who;
 - Allow others to review changes before applying them;
 - Let the changes be promoted to specific environments.

### What

 - A Git repository contains a desired state description of the landscape, with CI/CD and a review before merge regime;
 - Landscaper, an app that eliminates difference between desired and actual state of Helm releases in a Kubernetes cluster.

## Installation

### From source

You must have a working Go environment with [glide](https://github.com/Masterminds/glide) and `Make` installed.

From a terminal:
```shell
  cd $GOPATH
  mkdir -p src/github.com/eneco/
  cd !$
  git clone https://github.com/Eneco/landscaper.git
  cd landscaper
  make bootstrap build
```

Outputs the landscaper binary in `./build/landscaper`.

## Usage

Landscaper consists of a core API and a command line interface (CLI) that consumes this API. It takes as input a set of files that constitute the desired state. The CLI assumes that `kubectl` and `helm` have been setup!

These files contain (a) reference(s) to a Helm Chart and its configuration (Values).
Additionally, Landscaper receives settings (tokens, credentials) needed to query and modify a Kubernetes cluster via Helm/Tiller. Typically these settings are provided by the CI/CD system.

The CLI uses a command structure in the spirit of `git` et al. The main command is `landscaper apply` to apply a desired state.
The `apply` command accepts the following arguments:

        --chart-dir string   where the charts are stored (default "$HOME/.helm")
        --dir string         path to a folder that contains all the landscape desired state files; overrides LANDSCAPE_DIR (default ".")
        --dry-run            simulate the applying of the landscape. useful in merge requests
        --namespace string   namespace to apply the landscape to; overrides LANDSCAPE_NAMESPACE (default "acceptance")
        --no-prefix          disable prefixing release names
        --prefix string      prefix release names with this string instead of <namespace>; overrides LANDSCAPE_PREFIX
    -v, --verbose            be verbose

Instead of using arguments, environment variables can be used. When arguments are present, they override environment variables.
`--namespace` is used to isolate landscapes through Kubernetes namespaces.
Unless otherwise specified, Helm releases are prefixed with the same namespace string to avoid collisions, since Helm release names aren't namespaced.

Input desired state files are in YAML and contain the name that identifies the "component", a reference to a chart, configuration and optionally secrets.
Currently, secrets are handled by specifying the name in the YAML, e.g. `my-secret`, and having a matching environment variable available with the secret, e.g. `export MY_SECRET=Rumpelstiltskin`

## Basic Example

Create a directory with only a file `landscaped-mysql.yaml` that contains:

    name: landscaped-mysql
    release:
      chart: stable/mysql
      version: 0.1.0
    configuration:
      mysqlUser: landscaper
      mysqlPassword: demo

(or use the provided `/example/`).
From that directory, execute `landscaper apply --namespace example` and inspect the output.
On succes, `helm list` should output something like:

    NAME                        REVISION    UPDATED                     STATUS      CHART
    example-landscaped-mysql    1           Thu Dec  1 21:54:21 2016    DEPLOYED    mysql-0.2.1

Now use `kubectl get pod --namespace example` to obtain the pod name and status.
When the pod is `Running`, port forward with:

    kubectl port-forward --namespace example <your-pod> 3306

In another terminal, assuming the `mysql` client being installed, connect with the landscaped user/password:

    mysql --host localhost --protocol=TCP --user=landscaper --password=demo

## Example Use Case

We, at Eneco, have setup a git repository with the inputs to the landscaper. During CI, non-master branches are Landscaped `--dry-run` to validate the inputs. After a pull request is reviewed, the changes are merged into `master` after which the Landscaper applies the new desired state.

## Contributing

We'd love to accept your contributions! Please use GitHub pull requests: fork the repo, develop and test your code, submit a pull request. Thanks!
