FROM golang:1.23 AS builder
COPY . /go/src/github.com/skpr/k8s-mutate-nodeselector
WORKDIR /go/src/github.com/skpr/k8s-mutate-nodeselector
RUN CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o skpr-k8s-mutate-nodeselector github.com/skpr/k8s-mutate-nodeselector/cmd/skpr-k8s-mutate-nodeselector

FROM alpine:3.20
COPY --from=builder /go/src/github.com/skpr/k8s-mutate-nodeselector/skpr-k8s-mutate-nodeselector /usr/local/bin/skpr-k8s-mutate-nodeselector
RUN chmod +x /usr/local/bin/skpr-k8s-mutate-nodeselector
CMD ["/usr/local/bin/skpr-k8s-mutate-nodeselector"]
