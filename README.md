# Landscaper

[![Build Status](https://travis-ci.org/Eneco/landscaper.svg?branch=master)](https://travis-ci.org/Eneco/landscaper)
[![Go Report Card](https://goreportcard.com/badge/github.com/eneco/landscaper)](https://goreportcard.com/report/github.com/eneco/landscaper)
[![Go Doc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](https://godoc.org/github.com/Eneco/landscaper/pkg/landscaper)
[![Say Thanks!](https://img.shields.io/badge/Say%20Thanks-!-1EAEDB.svg)](https://saythanks.io/to/eneco)

Landscaper takes a set of Helm Chart references with values (a desired state), and realizes this in a Kubernetes cluster. The intended use case is to have this desired state under version control, and let Landscaper first test and then apply the state as part of the CI/CD stages.

The Landscaper project is an ongoing project in an early stage, opened due to demand in the community. Contributions and feedback are more than welcome!

 - [Introduction](#introduction)
 - [Usage](#usage)
 - [Example](#example)
 - [Example Use Case](#example-use-case)
 - [Contributing](#contributing)

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

Binaries are available [here](https://github.com/Eneco/landscaper/releases/); Docker images [here](https://hub.docker.com/r/eneco/landscaper/).
On macOS using Homebrew, a `brew install landscaper` should do.

### From source

You must have a working Go environment with [`dep`](https://github.com/golang/dep) and `GNU Make` installed.

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

    Usage:
      landscaper apply [files]... [flags]
    
    Flags:
          --azure-keyvault string         azure keyvault for fetching secrets. Azure credentials must be provided in the environment.
          --chart-dir string              (deprecated; use --helm-home) Helm home directory (default "$HOME/.helm")
          --config-override-file string   global configuration overrides. component specific environment overrides take precedence over this.
          --context string                the kube context to use. defaults to the current context
          --dir string                    (deprecated) path to a folder that contains all the landscape desired state files; overrides LANDSCAPE_DIR
          --disable stringSlice           Stages to be disabled. Available stages are create/update/delete.
          --dry-run                       simulate the applying of the landscape. useful in merge requests
          --helm-home string              Helm home directory (default "$HOME/.helm")
          --env string                    environment specifier. selects value overrides by environment.
          --loop                          keep landscape in sync forever
          --loop-interval duration        when running in a loop the interval between invocations (default 5m0s)
          --namespace string              namespace to apply the landscape to; overrides LANDSCAPE_NAMESPACE (default "default")
          --no-prefix                     disable prefixing release names
          --prefix string                 prefix release names with this string instead of <namespace>; overrides LANDSCAPE_PREFIX
          --tiller-namespace string       Tiller namespace for Helm (default "kube-system")
      -v, --verbose                       be verbose
          --wait                          wait for all resources to be ready
          --wait-timeout duration         interval to wait for all resources to be ready (default 5m0s)

Instead of using arguments, environment variables can be used. When arguments are present, they override environment variables.
`--namespace` is used to isolate landscapes through Kubernetes namespaces.
Unless otherwise specified, Helm releases are prefixed with the same namespace string to avoid collisions, since Helm release names aren't namespaced.
As of version 1.0.3, components can specify their namespace. This will override any provided global namespace.



Landscaper can also be run as a control loop that constantly watches the desired landscape and applies it to the cluster. With this you can deploy landscaper once in your cluster, pass it a reference to a landscape description and have Landscaper apply it whenever the landscape changes.

Connection to Tiller is made by setting up a port-forward to it's pod. However, when `$HELM_HOST` is defined with a "host:port" in it, a direct connection is made to that host and port instead.

### Azure Credentials
When using the `--azure-keyvault` argument, Azure Service Principal credentials must be available in the environment:

  - `AZURE_CLIENT_ID`
  - `AZURE_CLIENT_SECRET`
  - `AZURE_TENANT_ID`

The key vault DNS suffix defaults to the public cloud, `vault.azure.net`, but can be overridden with `AZURE_KEYVAULT_DNS_SUFFIX`.

### Desired State Files
Input desired state files are in YAML and contain the name that identifies the "component", a reference to a chart, configuration and optionally secrets.

    name: my-component
    release:
    	chart: "example/chart:0.1.0"
    	version: 0.1.0
    
    # This will become the .Values override for the chart deployment.
    configuration: 
      hostname: "example"
      url: "http://default.example.com"
      
    # These configurations can be selected with the '--env' flag 
    # and will override the above values. 
    environments: 
      dev:
        url: "http://dev.example.com"
      prod:
        url: "http://prod.example.com"
    
    # These secrets will be fetched, and instantiated as a Kubernetes 
    # secret whose name will be written to .Values.secretsRef
    secrets:     
    - my-secret
    - my-other-secret

Installing the above component with `default` prefix and `--env dev` will result in chart value overrides
  
    hostname: "example"
    url: "http://dev.example.com"
    secretsRef: 'default-my-component'
    
and a Kubernetes secret named `default-my-component` with the contents:

    
#### Secrets

Secrets can be provided as a list or as a map. If a list is provided then the same string is used for the key in the Kubernetes secret and to find the secret value.

```
...
secrets:
	- my-secret
	- my-other-secret
```

is realised as:

```
    my-secret: <value of MY_SECRET environment variable>
    my-other-secret: <value MY_OTHER_SECRET environment variable>
```

If a map is provided the key is the used as the Kubernetes secret key, and the value is used to fetch the secret value.

```
...
secrets:
	secret1: my-secret
	secret2: my-other-secret
```

is realised as:

```
    secret1: <value of MY_SECRET environment variable>
    secret2: <value MY_OTHER_SECRET environment variable>
```
    
By default the secrets are read from the environment with the string converted to `UPPER_SNAKE_CASE` e.g. `export MY_SECRET=Rumpelstiltskin`
    
### Global configuration override file

You can specify a global configuration override file with the `--config-override-file` argument. This will override chart and component defaults, but not environment specific configuration.

    # my-component.yaml
    name: my-component
    release:
      chart: "example/chart:0.1.0"
      version: 0.1.0
    
    # This will become the .Values override for the chart deployment.
    configuration:
      hostname: "example"
      url: "http://default.example.com"
    environments:
      env1:
        hostname: "env1"
    
 
    
    # global.yaml
    hostname: "global"
    url: "http://global.example.com"

Installing the above component with `--config-override-file global.yaml` and `--env env1` will result in chart value overrides
    
    hostname: "env1"
    url: "http://global.example.com"
    
### Secret Usage in Helm Charts
Secrets are made available as Kubernetes Secrets (as shown above). The helm chart needs to be setup to [use the secret in a pod](https://kubernetes.io/docs/concepts/configuration/secret/#using-secrets), where the secret name is made available by the landscaper as `.Values.secretsRef`. For example, as an environment variable:

    env:
    - name: MY_SECRET
      valueFrom:
        secretKeyRef:
          name: {{ .Values.secretsRef }}
          key: my-secret
          
 Or mounted volume:
 
    volumes:
    - name: my-secret
      secret:
        secretName: {{ .Values.secretsRef }}
        items:
        - key: my-secret
          path: secrets/my-secret
          mode: 511
          
## Example

An example is provided [here](./example).

## Example Use Case

We, at Eneco, have setup a git repository with the inputs to the landscaper. During CI, non-master branches are Landscaped `--dry-run` to validate the inputs. After a pull request is reviewed, the changes are merged into `master` after which the Landscaper applies the new desired state.

### Example disable a stage

Landscaper is build on philosphy of applying charts in three stages delete, update and create and in that order.

However it allows to disable stages during apply. So users can cover their corner cases.

e.g to disable delete stage

```
landscaper apply [files] --disable delete
```

and to disable delete and update

```
landscaper apply [files] --disable delete --disable update
```

## Compatibility

Landscaper uses both Helm and Kubernetes. The following Landscaper releases are built against the following versions:

| Landscaper | Helm  | Kubernetes |
|------------|-------|------------|
| 1.0.17     | 2.8.2 | 1.9        |
| 1.0.16     | 2.7.2 | 1.8        |
| 1.0.15     | 2.7.2 | 1.8        |
| 1.0.14     | 2.7.2 | 1.8        |
| 1.0.13     | 2.7.2 | 1.8        |
| 1.0.12     | 2.7.2 | 1.8        |
| 1.0.11     | 2.6.1 | 1.7        |
| 1.0.10     | 2.5.1 | 1.6        |
| 1.0.9      | 2.5.1 | 1.6        |
| 1.0.8      | 2.4.2 | 1.6        |
| 1.0.7      | 2.4.2 | 1.6        |
| 1.0.6      | 2.4.2 | 1.6        |
| 1.0.5      | 2.4.2 | 1.6        |
| 1.0.4      | 2.3.1 | 1.5        |
| 1.0.3      | 2.1.3 | 1.5        |
| 1.0.2      | 2.1.3 | 1.5        |
| 1.0.1      | 2.1.3 | 1.5        |
| 1.0.0      | 2.1.3 | 1.5        |

## Contributing

We'd love to accept your contributions! Please use GitHub pull requests: fork the repo, develop and test your code, [semantically commit](http://karma-runner.github.io/1.0/dev/git-commit-msg.html) (as of April 2017) and submit a pull request. Thanks!
