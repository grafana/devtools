# Builder image
FROM golang:1.13 AS builder

WORKDIR /build/

COPY ./cmd ./cmd
COPY ./pkg ./pkg
COPY ./go.mod ./go.sum ./

RUN go build \
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
