# builder image
FROM golang:1.9 as builder
WORKDIR /go/src/github.com/yacut/kubernetes-rbac-synchroniser
COPY . .
RUN go get -u github.com/prometheus/client_golang/prometheus/promhttp; \
	make test; make build;

# final image
FROM alpine:latest
COPY --from=builder /go/src/github.com/yacut/kubernetes-rbac-synchroniser/build/kubernetes-rbac-synchroniser /bin/kubernetes-rbac-synchroniser
ENTRYPOINT ["/bin/kubernetes-rbac-synchroniser"]
