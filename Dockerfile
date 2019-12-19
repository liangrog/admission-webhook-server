FROM alpine:3.10

RUN apk update --no-cache && apk add ca-certificates

COPY admitCtlr /admitCtlr

ENTRYPOINT ["/admitCtlr"]
