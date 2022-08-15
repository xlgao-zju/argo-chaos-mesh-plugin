FROM golang:1.17 AS builder

WORKDIR /go/src/github.com/xlgao-zju/argo-chaos-mesh-plugin
COPY . /go/src/github.com/xlgao-zju/argo-chaos-mesh-plugin

RUN USEVENDOR=yes make bin

FROM centos:7
COPY --from=builder /go/src/github.com/xlgao-zju/argo-chaos-mesh-plugin/bin/argo-chaos-mesh-plugin /bin/
