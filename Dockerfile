FROM golang:alpine AS builder
LABEL stage=builder
RUN apk add --no-cache gcc libc-dev git ca-certificates
WORKDIR /workspace
COPY main.go go.sum go.mod ./
COPY pkg ./pkg/
RUN CGO_ENABLED=0 GOOS=linux go build -a -o admitCtlr

FROM alpine AS final
WORKDIR /
COPY --from=builder /workspace/admitCtlr .
CMD [ "/admitCtlr" ]
