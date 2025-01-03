FROM golang:1.23-alpine as builder
RUN apk add --no-cache git ca-certificates curl && \
    update-ca-certificates

RUN git clone https://github.com/magefile/mage --depth 1 && \
    cd mage && \
    go run bootstrap.go

WORKDIR /go/src/app
COPY . .
RUN mage -v build

FROM alpine
RUN apk add --no-cache ca-certificates
COPY --from=builder /go/src/app/bin/ /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/grafana-ntfy"]
