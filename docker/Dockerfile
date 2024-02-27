FROM golang:1.21-alpine as build-stage

RUN apk --no-cache add \
    g++ \
    git \
    make \
    bash

ARG VERSION
ENV VERSION=${VERSION}

WORKDIR /src
COPY . .
RUN go build -o /src/bin/k8s-admission-webhook-drmax

FROM alpine:latest
RUN apk --no-cache add \
    ca-certificates
COPY --from=build-stage /src/bin/k8s-admission-webhook-drmax /usr/local/bin/k8s-admission-webhook-drmax
ENTRYPOINT ["/usr/local/bin/k8s-admission-webhook-drmax"]