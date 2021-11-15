FROM alpine:3.14.3

RUN apk add --no-cache ca-certificates

ADD ./flannel-operator /flannel-operator

ENTRYPOINT ["/flannel-operator"]
