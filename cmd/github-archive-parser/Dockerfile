# Builder image
FROM golang:1.13 AS builder

WORKDIR /build/

COPY ./cmd ./cmd
COPY ./pkg ./pkg
COPY ./go.mod ./go.sum ./

RUN go build \
  -o /build/github-archive-parser \
  ./cmd/github-archive-parser

# Runtime image

FROM debian:buster-slim AS runtime

ENV DEBIAN_FRONTEND noninteractive
RUN apt-get update && \
  apt-get install -y ca-certificates

COPY --from=builder /build/github-archive-parser /bin/github-archive-parser

ENTRYPOINT [ "/bin/github-archive-parser" ]
CMD [ "-help" ]
