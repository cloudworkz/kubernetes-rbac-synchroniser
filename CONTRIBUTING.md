## Contributing guidelines

### Prepare

- Install minikube: `brew cask install virtualbox minikube`
- Install go: `brew install go`
- Start local k8s cluster: `minikube start --extra-config=apiserver.Authorization.Mode=RBAC`
- Run rbac sync: `kubectl run kubernetes-rbac-synchroniser --image=yacut/kubernetes-rbac-synchroniser --replicas=1 --port=8080`
