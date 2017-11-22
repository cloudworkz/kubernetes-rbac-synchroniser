# kubernetes-rbac-synchroniser
[![Docker Repository on Quay](https://quay.io/repository/yacut/kubernetes-rbac-synchroniser/status "Docker Repository on Quay")](https://quay.io/repository/yacut/kubernetes-rbac-synchroniser)
[![Docker Pulls](https://img.shields.io/docker/pulls/yacut/kubernetes-rbac-synchroniser.svg?maxAge=604800)](https://hub.docker.com/r/yacut/kubernetes-rbac-synchroniser)
[![Go Report Card](https://goreportcard.com/badge/github.com/yacut/kubernetes-rbac-synchroniser)](https://goreportcard.com/report/github.com/yacut/kubernetes-rbac-synchroniser)


## Requirements

- The service account's private key file: **-credentials-file-path** flag
- The email of the user with permissions to access the Admin APIs:  **-config-subject** flag

> see guide: https://developers.google.com/admin-sdk/directory/v1/guides/delegation

### Links

- https://github.com/kubernetes/client-go
- https://github.com/prometheus/client_golang
