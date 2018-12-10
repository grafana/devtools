FROM debian:stretch-slim

RUN apt-get update && apt-get install -y ca-certificates

COPY github-archive-parser /bin/github-archive-parser
ENTRYPOINT [ "/bin/github-archive-parser" ]
