FROM alpine:3.12.0

RUN apk add --no-cache ca-certificates

ADD ./flannel-operator /flannel-operator

ENTRYPOINT ["/flannel-operator"]
