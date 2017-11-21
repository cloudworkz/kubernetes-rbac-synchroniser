# kubernetes-rbac-synchroniser
[![Docker Repository on Quay](https://quay.io/repository/yacut/kubernetes-rbac-synchroniser/status "Docker Repository on Quay")](https://quay.io/repository/yacut/kubernetes-rbac-synchroniser)
[![Docker Pulls](https://img.shields.io/docker/pulls/yacut/kubernetes-rbac-synchroniser.svg?maxAge=604800)](https://hub.docker.com/r/yacut/kubernetes-rbac-synchroniser)
[![Go Report Card](https://goreportcard.com/badge/github.com/yacut/kubernetes-rbac-synchroniser)](https://goreportcard.com/report/github.com/yacut/kubernetes-rbac-synchroniser)


## Preparation

- Create the google group api credentials json file: `.credentials/client_secret.json`
- Run the app and generate the access token
- Run the app again with the `-token` flag

### Links

- https://github.com/kubernetes/client-go
- https://github.com/prometheus/client_golang
