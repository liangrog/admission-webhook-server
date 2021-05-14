FROM golang:1.14 AS builder

WORKDIR /data
COPY . .
RUN GOPATH=/gopath CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o target/admitCtlr main.go

FROM alpine:3.10

# Install dependencies
RUN apk update --no-cache && apk add ca-certificates

# Copy go binary
RUN mkdir -p /opt/app
WORKDIR /opt/app
COPY --from=builder /data/target/admitCtlr /opt/app/admitCtlr
RUN chmod +x /opt/app/admitCtlr
CMD ["/opt/app/admitCtlr"]