FROM golang:1.9.2-alpine as code

RUN apk update && apk add git findutils \
    && go get -u github.com/vmware/govmomi/govc \
    && apk del git \
    && rm -rf /var/cache/apk/*

FROM alpine:3.8

RUN apk upgrade --update --no-cache \
    && mkdir /.govmomi \
    && chown nobody:nobody /.govmomi

USER nobody

ENV GOVC_URL setme
ENV GOVC_INSECURE true

COPY --from=code /go/bin/govc /usr/local/bin/govc

COPY build/_output/bin/vspherecompute /usr/local/bin/vspherecompute