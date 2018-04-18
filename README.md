## kubernetes-rbac-synchroniser
[![license](https://img.shields.io/github/license/google-cloud-tools/kubernetes-rbac-synchroniser.svg?maxAge=604800)](https://github.com/google-cloud-tools/kubernetes-rbac-synchroniser)
[![Docker Repository on Quay](https://quay.io/repository/google-cloud-tools/kubernetes-rbac-synchroniser/status "Docker Repository on Quay")](https://quay.io/repository/google-cloud-tools/kubernetes-rbac-synchroniser)
[![Docker Pulls](https://img.shields.io/docker/pulls/google-cloud-tools/kubernetes-rbac-synchroniser.svg?maxAge=604800)](https://hub.docker.com/r/google-cloud-tools/kubernetes-rbac-synchroniser)
[![Go Report Card](https://goreportcard.com/badge/github.com/google-cloud-tools/kubernetes-rbac-synchroniser)](https://goreportcard.com/report/github.com/google-cloud-tools/kubernetes-rbac-synchroniser)

### What It Does

RBAC Synchroniser pulls a Google Group, extracts Google Group Member Emails and updates the Kubernetes RoleBinding in the given namespace.

[![graph](https://raw.githubusercontent.com/google-cloud-tools/kubernetes-rbac-synchroniser/master/graph.png)](https://raw.githubusercontent.com/google-cloud-tools/kubernetes-rbac-synchroniser/master/graph.png)

### Requirements

- The service account's private key file: **-config-file-path** flag
- The email of the user with permissions to access the Admin APIs:  **-google-admin-email** flag

> see guide: https://developers.google.com/admin-sdk/directory/v1/guides/delegation

- The Google Group list per Kubernetes namespace: **-namespace-group** flag
- Configure Minimal GKE IAM permissions for each Google Group: `gcloud beta iam roles create minimal_gke_role --project my_project --title "Container Engine Minimal" --description "Minimal GKE Role which allows 'gcloud container clusters get-credentials' command" --permissions "container.apiServices.get,container.apiServices.list,container.clusters.get,container.clusters.getCredentials"`

> see: https://stackoverflow.com/questions/45945074/iam-and-rbac-conflicts-on-google-cloud-container-engine-gke/45945239#45945239

### Flags

| Flag                 | Description                                              | Defalut     |
| :------------------- | :------------------------------------------------------- |:----------- |
| -cluster-role-name   | The cluster role name with permissions.                  | "view"      |
| -config-file-path    | The Path to the Service Account's Private Key file.      |             |
| -google-admin-email  | The Google Admin Email.                                  |             |
| -fake-group-response | Fake Google Admin API Response.                          |             |
| -namespace-group     | The group and namespace. May be used multiple times.     |             |
| -in-cluster-config   | Use in cluster kubeconfig.                               | true        |
| -kubeconfig          | Absolute path to the kubeconfig file.                    |             |
| -listen-address      | The address to listen on for HTTP requests.              | ":8080"     |
| -rolebinding-name    | The role binding name per namespace.                     | "developer" |
| -update-interval     | Update interval in seconds.                              | 15m0s       |
| -log-json            | Log as JSON instead of the default ASCII formatter.      | false       |

### Prometheus metrics

- **rbac_synchroniser_success**: Cumulative number of role update operations.
- **rbac_synchroniser_errors**: Cumulative number of errors during role update operations.

### Examples

[https://github.com/google-cloud-tools/kubernetes-rbac-synchroniser/tree/master/examples](https://github.com/google-cloud-tools/kubernetes-rbac-synchroniser/tree/master/examples)

### Links

- https://developers.google.com/admin-sdk/directory/v1/guides/delegation
- https://developers.google.com/admin-sdk/directory/v1/guides/manage-group-members
- https://github.com/kubernetes/client-go
- https://github.com/prometheus/client_golang
