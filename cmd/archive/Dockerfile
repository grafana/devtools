FROM debian:stretch-slim

RUN apt-get update && apt-get install -y ca-certificates

COPY archive /bin/archive
ENTRYPOINT [ "/bin/archive" ]
