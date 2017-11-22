## Contributing guidelines

### HowTo

- Install minikube: `brew cask install virtualbox minikube`
- Install go: `brew install go`
- Install dependencies: `make install`
- Start local k8s cluster: `minikube start --extra-config=apiserver.Authorization.Mode=RBAC`
- Run rbac sync: `kubectl apply -f example/`
- Run localy: `go run kubernetes-rbac-synchroniser.go -h`
