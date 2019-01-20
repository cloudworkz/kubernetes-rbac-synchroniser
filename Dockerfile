# builder image
FROM golang:1.11.4 as builder
WORKDIR /go/src/github.com/yacut/kubernetes-rbac-synchroniser
COPY . .
RUN make install; \
	CGO_ENABLED=0 GOOS=linux go build -o build/kubernetes-rbac-synchroniser; \
  curl -o ca-certificates.crt https://curl.haxx.se/ca/cacert.pem;

# final image
FROM scratch
COPY --from=builder /go/src/github.com/yacut/kubernetes-rbac-synchroniser/build/kubernetes-rbac-synchroniser /bin/kubernetes-rbac-synchroniser
COPY --from=builder /go/src/github.com/yacut/kubernetes-rbac-synchroniser/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
ENTRYPOINT ["/bin/kubernetes-rbac-synchroniser"]
