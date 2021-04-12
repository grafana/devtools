# Builder image
FROM golang:1.12 AS builder

WORKDIR /build/

COPY ./cmd ./cmd
COPY ./pkg ./pkg
COPY ./vendor ./vendor
COPY ./go.mod ./go.sum ./

RUN go build -mod=vendor \
  -o /build/bots \
  ./cmd/bots

# Runtime image

FROM debian:buster-slim AS runtime

ENV DEBIAN_FRONTEND noninteractive
RUN apt-get update && \
  apt-get install -y ca-certificates

COPY --from=builder /build/bots /bin/bots

ENTRYPOINT [ "/bin/bots" ]
CMD [ "-help" ]
