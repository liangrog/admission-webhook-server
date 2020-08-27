FROM alpine:3.10

# Install dependencies
RUN apk update --no-cache && apk add ca-certificates

# Copy go binary
RUN mkdir -p /opt/app
WORKDIR /opt/app
COPY target/admitCtlr /opt/app/admitCtlr
RUN chmod +x /opt/app/admitCtlr
CMD ["/opt/app/admitCtlr"]