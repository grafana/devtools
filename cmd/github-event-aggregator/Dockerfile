# Builder image
FROM golang:1.13 AS builder

WORKDIR /build/

COPY ./cmd ./cmd
COPY ./pkg ./pkg
COPY ./vendor ./vendor
COPY ./go.mod ./go.sum ./

RUN go build -mod=vendor \
    -o /build/github-event-aggregator \
    ./cmd/github-event-aggregator

# Runtime image

FROM debian:buster-slim AS runtime

ENV DEBIAN_FRONTEND noninteractive
RUN apt-get update && \
  apt-get install -y ca-certificates

COPY --from=builder /build/github-event-aggregator /bin/github-event-aggregator

ENTRYPOINT [ "/bin/github-event-aggregator" ]
CMD [ "-help" ]
