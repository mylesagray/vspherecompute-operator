FROM golang:1.9.2-alpine as code

RUN apk update && apk add git findutils \
    && go get -u github.com/vmware/govmomi/govc \
    && apk del git \
    && rm -rf /var/cache/apk/*

FROM alpine:3.8

RUN apk upgrade --update --no-cache

USER nobody

ENV GOVC_URL setme
ENV GOVC_INSECURE true

COPY --from=code /go/bin/govc /govc

ADD build/_output/bin/vspherecompute /usr/local/bin/vspherecompute