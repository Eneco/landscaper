# Landscaper

Landscaper eliminates difference between desired and actual state of Helm releases in a Kubernetes cluster.

## Introduction

### Why

 - The set of applications (e.g. connectors, processors) in the Kubernetes cluster is potentially large and complex.
 - Manually inspecting and administrating them, and keeping different environments (prod, acceptance) in sync is laborious and difficult.
 - Cooperating in this shared cluster increases the complexity.

### How

 - Have a blue print of what the landscape (apps in the cluster) looks like;
 - Keep track of changes: when, what, why and by who;
 - Allow others to review changes before applying them;
 - Let the changes be promoted to specific environments.

### What

 - A Git repository contains a desired state description of the landscape, with CI/CD;
 - Landscaper, an app that eliminates difference between desired and actual state of Helm releases in a Kubernetes cluster.

## Usage

Landscaper takes as input a set of files that constitute the desired state.
These files contain (a) reference(s) to a Helm Chart and its configuration (Values).
Additionally, it receives settings needed to query and modify a Kubernetes cluster via Helm/Tiller.
