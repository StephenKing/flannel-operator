FROM alpine:3.4

RUN apk update && apk upgrade && apk add ca-certificates

ADD operator /bin/operator

ENTRYPOINT ["/bin/operator"]
