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

## Example

Create a file `rt-wind-exporter.yaml` with:

    name: rtwind-exporter-impala
    release:
      chart: eet/exporter-impala:0.1.3
      version: 1.0.1
    configuration:
      FTPServer: engdapp07201.engd.local
      JobSchedule: 0/1 * * * ?
    secrets:
      - ftp-password

Add an environment variable with the password: `export FTP_PASSWORD=Rumpelstiltskin`.

In the Chart template there are lines like:


      - name: "{{ .Values.Key }}_FTP_PASS"
        valueFrom:
          secretKeyRef:
            name: {{ .Values.SecretsRef }}
            key: ftp-password
      - name: "{{ .Values.Key }}_FTP_SERVER"
        value: {{ .Values.FTPServer }}

Which receive the landscaped FTPServer address and provided password.

Now do a `landscaper apply --dry-run` to give it a try, or omit `dry-run` for the real thing.

## Example Use Case

We, at Eneco, have setup a git repository with the inputs to the landscaper. During CI, non-master branches are Landscaped `--dry-run` to validate the inputs. After a pull request is reviewed, the changes are merged into `master` after which the Landscaper applies the new desired state.
