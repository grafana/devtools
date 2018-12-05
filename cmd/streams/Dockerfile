FROM debian:stretch-slim

RUN apt-get update && apt-get install -y ca-certificates

COPY streams /bin/streams
ENTRYPOINT [ "/bin/streams" ]
