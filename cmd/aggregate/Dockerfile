FROM debian:stretch-slim

RUN apt-get update && apt-get install -y ca-certificates

COPY aggregate /bin/aggregate
ENTRYPOINT [ "/bin/aggregate" ]
