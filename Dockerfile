# builder image
FROM golang:1.9 as builder
WORKDIR /go/src/github.com/yacut/kubernetes-rbac-synchroniser
COPY . .
RUN go get -u github.com/prometheus/client_golang/prometheus/promhttp; \
	CGO_ENABLED=0 GOOS=linux go build -o build/kubernetes-rbac-synchroniser;

# final image
FROM scratch
COPY --from=builder /go/src/github.com/yacut/kubernetes-rbac-synchroniser/build/kubernetes-rbac-synchroniser /bin/kubernetes-rbac-synchroniser
ENTRYPOINT ["/bin/kubernetes-rbac-synchroniser"]
