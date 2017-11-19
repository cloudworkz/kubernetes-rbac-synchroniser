FROM alpine

COPY ./bin/kubernetes-rbac-synchroniser /kubernetes-rbac-synchroniser

CMD ["/kubernetes-rbac-synchroniser"]
