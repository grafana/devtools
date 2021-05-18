# Builder image
FROM golang:1.13 AS builder

WORKDIR /build/

COPY ./cmd ./cmd
COPY ./pkg ./pkg
COPY ./vendor ./vendor
COPY ./go.mod ./go.sum ./

RUN go build -mod=vendor \
  -o /build/local-gharchive-server \
  ./cmd/local-gharchive-server

# Runtime image

FROM debian:buster-slim AS runtime

ENV DEBIAN_FRONTEND noninteractive
RUN apt-get update && \
  apt-get install -y ca-certificates

COPY --from=builder /build/local-gharchive-server /bin/local-gharchive-server

ENTRYPOINT [ "/bin/local-gharchive-server" ]
CMD [ "-help" ]
