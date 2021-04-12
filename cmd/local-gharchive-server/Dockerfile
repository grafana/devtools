FROM debian:buster-slim

RUN apt-get update && apt-get install -y ca-certificates

COPY local-gharchive-server /bin/local-gharchive-server
ENTRYPOINT [ "/bin/local-gharchive-server" ]
